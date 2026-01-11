(* MixOS Build Executor Service
   Deterministic build execution with sandboxing *)

open Lwt.Infix

(* Build request/response types *)
type build_request = {
  package_name: string;
  source_path: string;
  output_path: string;
  env: (string * string) list;
  build_args: string list;
} [@@deriving yojson]

type build_response = {
  success: bool;
  error: string option;
  artifact_path: string;
  artifact_hash: string;
  logs: string list;
} [@@deriving yojson]

type build_status = {
  job_id: string;
  package_name: string;
  status: string; (* "pending", "building", "success", "failed" *)
  progress: int;
  started_at: float;
  finished_at: float option;
} [@@deriving yojson]

(* Build job *)
type build_job = {
  id: string;
  request: build_request;
  mutable status: string;
  mutable logs: string list;
  mutable started_at: float;
  mutable finished_at: float option;
}

(* State *)
let jobs : (string, build_job) Hashtbl.t = Hashtbl.create 100
let job_counter = ref 0

let store_path = "/store"
let build_path = "/tmp/mixos-build"

(* Logging *)
let log level msg =
  let timestamp = Unix.gettimeofday () in
  Printf.printf "[%.3f] [builder] [%s] %s\n%!" timestamp level msg

(* Generate job ID *)
let next_job_id () =
  incr job_counter;
  Printf.sprintf "build-%d-%d" (int_of_float (Unix.gettimeofday ())) !job_counter

(* Calculate hash of directory *)
let hash_directory path =
  (* Simplified: hash all files concatenated *)
  let cmd = Printf.sprintf "find %s -type f -exec sha256sum {} \\; | sort | sha256sum | cut -d' ' -f1" path in
  let ic = Unix.open_process_in cmd in
  let hash = input_line ic in
  ignore (Unix.close_process_in ic);
  String.trim hash

(* Setup deterministic build environment *)
let setup_build_env job =
  log "INFO" (Printf.sprintf "Setting up build environment for %s" job.request.package_name);
  
  (* Create isolated build directory *)
  let build_dir = Filename.concat build_path job.id in
  (try Unix.mkdir build_path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  Unix.mkdir build_dir 0o755;
  
  (* Copy source to build directory *)
  let src_dir = Filename.concat build_dir "src" in
  let cmd = Printf.sprintf "cp -r %s %s" job.request.source_path src_dir in
  let _ = Unix.system cmd in
  
  Lwt.return build_dir

(* Execute build *)
let execute_build job build_dir =
  log "INFO" (Printf.sprintf "Executing build for %s" job.request.package_name);
  job.status <- "building";
  job.started_at <- Unix.gettimeofday ();
  
  let src_dir = Filename.concat build_dir "src" in
  let out_dir = Filename.concat build_dir "out" in
  Unix.mkdir out_dir 0o755;
  
  (* Build environment variables for determinism *)
  let base_env = [
    ("HOME", build_dir);
    ("TMPDIR", Filename.concat build_dir "tmp");
    ("SOURCE_DATE_EPOCH", "1");
    ("TZ", "UTC");
    ("LC_ALL", "C");
    ("PATH", "/bin:/sbin:/usr/bin:/usr/sbin");
  ] in
  let env = base_env @ job.request.env in
  let env_str = String.concat " " (List.map (fun (k, v) -> Printf.sprintf "%s=%s" k v) env) in
  
  (* Create tmp directory *)
  Unix.mkdir (Filename.concat build_dir "tmp") 0o755;
  
  (* Execute build command *)
  let build_script = Filename.concat src_dir "build.sh" in
  let cmd = if Sys.file_exists build_script then
    Printf.sprintf "cd %s && env -i %s sh build.sh 2>&1" src_dir env_str
  else
    Printf.sprintf "cd %s && env -i %s make 2>&1" src_dir env_str
  in
  
  let ic = Unix.open_process_in cmd in
  let rec read_logs acc =
    try
      let line = input_line ic in
      job.logs <- job.logs @ [line];
      read_logs (line :: acc)
    with End_of_file -> List.rev acc
  in
  let logs = read_logs [] in
  let status = Unix.close_process_in ic in
  
  match status with
  | Unix.WEXITED 0 ->
    (* Calculate output hash *)
    let hash = hash_directory out_dir in
    
    (* Move to store *)
    let store_dir = Filename.concat store_path hash in
    let cmd = Printf.sprintf "mv %s %s" out_dir store_dir in
    let _ = Unix.system cmd in
    
    job.status <- "success";
    job.finished_at <- Some (Unix.gettimeofday ());
    log "INFO" (Printf.sprintf "Build successful: %s -> %s" job.request.package_name hash);
    
    Lwt.return {
      success = true;
      error = None;
      artifact_path = store_dir;
      artifact_hash = hash;
      logs = logs;
    }
    
  | _ ->
    job.status <- "failed";
    job.finished_at <- Some (Unix.gettimeofday ());
    log "ERROR" (Printf.sprintf "Build failed: %s" job.request.package_name);
    
    Lwt.return {
      success = false;
      error = Some "Build failed";
      artifact_path = "";
      artifact_hash = "";
      logs = logs;
    }

(* Cleanup build directory *)
let cleanup_build build_dir =
  let cmd = Printf.sprintf "rm -rf %s" build_dir in
  let _ = Unix.system cmd in
  Lwt.return_unit

(* Handle build request *)
let handle_build_request req =
  let job_id = next_job_id () in
  let job = {
    id = job_id;
    request = req;
    status = "pending";
    logs = [];
    started_at = 0.0;
    finished_at = None;
  } in
  Hashtbl.add jobs job_id job;
  
  log "INFO" (Printf.sprintf "Build job created: %s for %s" job_id req.package_name);
  
  (* Execute build asynchronously *)
  let%lwt build_dir = setup_build_env job in
  let%lwt response = execute_build job build_dir in
  let%lwt () = cleanup_build build_dir in
  
  Lwt.return response

(* Get build status *)
let get_build_status job_id =
  match Hashtbl.find_opt jobs job_id with
  | Some job -> Some {
      job_id = job.id;
      package_name = job.request.package_name;
      status = job.status;
      progress = (if job.status = "success" then 100 else if job.status = "building" then 50 else 0);
      started_at = job.started_at;
      finished_at = job.finished_at;
    }
  | None -> None

(* IPC types *)
type message_type = Request | Response | Event | Stream [@@deriving yojson]

type ipc_message = {
  version: int;
  msg_type: message_type;
  msg_id: int64;
  source: string;
  target: string;
  method_name: string; [@key "method"]
  payload: string;
  timestamp: int64;
  ipc_error: string option; [@key "error"]
} [@@deriving yojson]

(* IPC helpers *)
let read_message fd =
  let len_buf = Bytes.create 4 in
  Lwt_unix.read fd len_buf 0 4 >>= fun n ->
  if n < 4 then Lwt.return_none
  else begin
    let len = 
      (Char.code (Bytes.get len_buf 0) lsl 24) lor
      (Char.code (Bytes.get len_buf 1) lsl 16) lor
      (Char.code (Bytes.get len_buf 2) lsl 8) lor
      (Char.code (Bytes.get len_buf 3))
    in
    let msg_buf = Bytes.create len in
    Lwt_unix.read fd msg_buf 0 len >>= fun n ->
    if n < len then Lwt.return_none
    else
      match ipc_message_of_yojson (Yojson.Safe.from_string (Bytes.to_string msg_buf)) with
      | Ok msg -> Lwt.return_some msg
      | Error _ -> Lwt.return_none
  end

let write_message fd msg =
  let json_str = Yojson.Safe.to_string (ipc_message_to_yojson msg) in
  let len = String.length json_str in
  let len_buf = Bytes.create 4 in
  Bytes.set len_buf 0 (Char.chr ((len lsr 24) land 0xff));
  Bytes.set len_buf 1 (Char.chr ((len lsr 16) land 0xff));
  Bytes.set len_buf 2 (Char.chr ((len lsr 8) land 0xff));
  Bytes.set len_buf 3 (Char.chr (len land 0xff));
  Lwt_unix.write fd len_buf 0 4 >>= fun _ ->
  Lwt_unix.write_string fd json_str 0 len >>= fun _ ->
  Lwt.return_unit

(* Connect to broker *)
let connect_broker () =
  let socket = Lwt_unix.socket Unix.PF_UNIX Unix.SOCK_STREAM 0 in
  let addr = Unix.ADDR_UNIX "/run/mixos/ipc.sock" in
  Lwt_unix.connect socket addr >>= fun () ->
  
  let register_msg = {
    version = 1;
    msg_type = Request;
    msg_id = 1L;
    source = "builder";
    target = "broker";
    method_name = "register";
    payload = {|{"service": "builder", "type": "core"}|};
    timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
    ipc_error = None;
  } in
  write_message socket register_msg >>= fun () ->
  read_message socket >>= fun _ ->
  log "INFO" "Registered with broker";
  Lwt.return socket

(* Message loop *)
let message_loop socket =
  let rec loop () =
    read_message socket >>= function
    | None ->
      log "ERROR" "Broker connection lost";
      Lwt.return_unit
    | Some msg when msg.target = "builder" ->
      begin match msg.method_name with
      | "build" ->
        begin match build_request_of_yojson (Yojson.Safe.from_string msg.payload) with
        | Ok req ->
          let%lwt response = handle_build_request req in
          let response_msg = {
            version = 1;
            msg_type = Response;
            msg_id = msg.msg_id;
            source = "builder";
            target = msg.source;
            method_name = "build";
            payload = Yojson.Safe.to_string (build_response_to_yojson response);
            timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
            ipc_error = None;
          } in
          write_message socket response_msg
        | Error e ->
          log "ERROR" (Printf.sprintf "Invalid build request: %s" e);
          Lwt.return_unit
        end
      | "status" ->
        let job_id = msg.payload in
        let status = get_build_status job_id in
        let payload = match status with
          | Some s -> Yojson.Safe.to_string (build_status_to_yojson s)
          | None -> {|{"error": "Job not found"}|}
        in
        let response_msg = {
          version = 1;
          msg_type = Response;
          msg_id = msg.msg_id;
          source = "builder";
          target = msg.source;
          method_name = "status";
          payload;
          timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
          ipc_error = None;
        } in
        write_message socket response_msg
      | _ ->
        log "WARN" (Printf.sprintf "Unknown method: %s" msg.method_name);
        Lwt.return_unit
      end >>= loop
    | Some _ -> loop ()
  in
  loop ()

(* Main *)
let main () =
  log "INFO" "MixOS Build Executor starting...";
  
  (try Unix.mkdir store_path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  (try Unix.mkdir build_path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  
  let rec connect_with_retry attempts =
    if attempts <= 0 then Lwt.fail_with "Connection failed"
    else
      Lwt.catch
        (fun () -> connect_broker ())
        (fun _ ->
          log "WARN" "Broker not ready, retrying...";
          Lwt_unix.sleep 1.0 >>= fun () ->
          connect_with_retry (attempts - 1))
  in
  
  connect_with_retry 30 >>= fun socket ->
  log "INFO" "Build Executor ready";
  message_loop socket

let () = Lwt_main.run (main ())
