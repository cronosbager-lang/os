(* MixOS Package Manager Service
   Handles package installation, removal, and queries *)

open Lwt.Infix

(* Package metadata *)
type package_info = {
  name: string;
  version: string;
  description: string;
  dependencies: string list;
  size: int64;
  hash: string;
  installed: bool;
} [@@deriving yojson]

type package_request = {
  action: string;
  packages: string list;
  options: (string * string) list;
} [@@deriving yojson]

type package_response = {
  success: bool;
  error: string option;
  packages: package_info list;
} [@@deriving yojson]

(* Store paths *)
let store_path = "/store"
let db_path = "/var/lib/mixos/pkgdb"

(* Package database (in-memory for now) *)
let installed_packages : (string, package_info) Hashtbl.t = Hashtbl.create 100

(* Logging *)
let log level msg =
  let timestamp = Unix.gettimeofday () in
  Printf.printf "[%.3f] [pkgmgr] [%s] %s\n%!" timestamp level msg

(* Calculate content hash *)
let hash_file path =
  let ic = open_in_bin path in
  let len = in_channel_length ic in
  let content = really_input_string ic len in
  close_in ic;
  Digestif.SHA256.(to_hex (digest_string content))

(* Install package to store *)
let install_package pkg_path =
  log "INFO" (Printf.sprintf "Installing package from %s" pkg_path);
  
  (* Calculate hash *)
  let hash = hash_file pkg_path in
  let store_dir = Filename.concat store_path hash in
  
  (* Check if already in store *)
  if Sys.file_exists store_dir then begin
    log "INFO" (Printf.sprintf "Package already in store: %s" hash);
    Lwt.return_ok hash
  end else begin
    (* Create store directory *)
    Unix.mkdir store_dir 0o755;
    
    (* Extract package (simplified - just copy for now) *)
    let dest = Filename.concat store_dir "package" in
    let cmd = Printf.sprintf "cp -r %s %s" pkg_path dest in
    let _ = Unix.system cmd in
    
    log "INFO" (Printf.sprintf "Package installed to store: %s" hash);
    Lwt.return_ok hash
  end

(* Remove package from store *)
let remove_package hash =
  log "INFO" (Printf.sprintf "Removing package: %s" hash);
  let store_dir = Filename.concat store_path hash in
  if Sys.file_exists store_dir then begin
    let cmd = Printf.sprintf "rm -rf %s" store_dir in
    let _ = Unix.system cmd in
    Lwt.return_ok ()
  end else
    Lwt.return_error "Package not found in store"

(* Query package info *)
let query_package name =
  match Hashtbl.find_opt installed_packages name with
  | Some pkg -> Lwt.return_some pkg
  | None -> Lwt.return_none

(* List all installed packages *)
let list_packages () =
  Hashtbl.fold (fun _ pkg acc -> pkg :: acc) installed_packages []

(* Handle package request *)
let handle_request req =
  match req.action with
  | "install" ->
    log "INFO" (Printf.sprintf "Install request: %s" (String.concat ", " req.packages));
    (* For each package, install to store *)
    let%lwt results = Lwt_list.map_s (fun pkg_name ->
      (* In real implementation, would fetch from repository *)
      let pkg_info = {
        name = pkg_name;
        version = "1.0.0";
        description = "Package " ^ pkg_name;
        dependencies = [];
        size = 0L;
        hash = "";
        installed = true;
      } in
      Hashtbl.replace installed_packages pkg_name pkg_info;
      Lwt.return pkg_info
    ) req.packages in
    Lwt.return { success = true; error = None; packages = results }
    
  | "remove" ->
    log "INFO" (Printf.sprintf "Remove request: %s" (String.concat ", " req.packages));
    List.iter (fun name -> Hashtbl.remove installed_packages name) req.packages;
    Lwt.return { success = true; error = None; packages = [] }
    
  | "query" ->
    log "INFO" (Printf.sprintf "Query request: %s" (String.concat ", " req.packages));
    let%lwt results = Lwt_list.filter_map_s query_package req.packages in
    Lwt.return { success = true; error = None; packages = results }
    
  | "list" ->
    log "INFO" "List request";
    let packages = list_packages () in
    Lwt.return { success = true; error = None; packages }
    
  | action ->
    log "WARN" (Printf.sprintf "Unknown action: %s" action);
    Lwt.return { success = false; error = Some ("Unknown action: " ^ action); packages = [] }

(* IPC message handling *)
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

(* Read/write helpers *)
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
      let json_str = Bytes.to_string msg_buf in
      match ipc_message_of_yojson (Yojson.Safe.from_string json_str) with
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
  
  (* Register with broker *)
  let register_msg = {
    version = 1;
    msg_type = Request;
    msg_id = 1L;
    source = "pkgmgr";
    target = "broker";
    method_name = "register";
    payload = {|{"service": "pkgmgr", "type": "core"}|};
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
    | Some msg when msg.target = "pkgmgr" ->
      log "DEBUG" (Printf.sprintf "Received: %s" msg.method_name);
      begin match msg.method_name with
      | "package" ->
        begin match package_request_of_yojson (Yojson.Safe.from_string msg.payload) with
        | Ok req ->
          let%lwt response = handle_request req in
          let response_msg = {
            version = 1;
            msg_type = Response;
            msg_id = msg.msg_id;
            source = "pkgmgr";
            target = msg.source;
            method_name = "package";
            payload = Yojson.Safe.to_string (package_response_to_yojson response);
            timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
            ipc_error = None;
          } in
          write_message socket response_msg
        | Error e ->
          log "ERROR" (Printf.sprintf "Invalid request: %s" e);
          Lwt.return_unit
        end
      | _ ->
        log "WARN" (Printf.sprintf "Unknown method: %s" msg.method_name);
        Lwt.return_unit
      end >>= loop
    | Some _ -> loop ()
  in
  loop ()

(* Main *)
let main () =
  log "INFO" "MixOS Package Manager starting...";
  
  (* Create directories *)
  (try Unix.mkdir store_path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  (try Unix.mkdir db_path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  
  (* Connect to broker with retry *)
  let rec connect_with_retry attempts =
    if attempts <= 0 then begin
      log "ERROR" "Failed to connect to broker";
      Lwt.fail_with "Connection failed"
    end else begin
      Lwt.catch
        (fun () -> connect_broker ())
        (fun _ ->
          log "WARN" "Broker not ready, retrying...";
          Lwt_unix.sleep 1.0 >>= fun () ->
          connect_with_retry (attempts - 1))
    end
  in
  
  connect_with_retry 30 >>= fun socket ->
  log "INFO" "Package Manager ready";
  message_loop socket

let () = Lwt_main.run (main ())
