(* MixOS Artifact Cache Service
   Content-addressable storage for build artifacts *)

open Lwt.Infix

(* Cache entry *)
type cache_entry = {
  key: string;
  hash: string;
  size: int64;
  created_at: float;
  accessed_at: float;
  ttl: int64; (* 0 = never expire *)
} [@@deriving yojson]

(* Request/Response types *)
type cache_request = {
  action: string; (* "get", "put", "delete", "gc", "stats" *)
  key: string;
  value: string option; (* base64 encoded for binary *)
  ttl: int64;
} [@@deriving yojson]

type cache_response = {
  success: bool;
  error: string option;
  value: string option;
  hit: bool;
  stats: cache_stats option;
} [@@deriving yojson]

and cache_stats = {
  total_entries: int;
  total_size: int64;
  hit_count: int64;
  miss_count: int64;
} [@@deriving yojson]

(* Cache state *)
let cache_dir = "/store/cache"
let entries : (string, cache_entry) Hashtbl.t = Hashtbl.create 1000
let hit_count = ref 0L
let miss_count = ref 0L

(* Logging *)
let log level msg =
  let timestamp = Unix.gettimeofday () in
  Printf.printf "[%.3f] [cache] [%s] %s\n%!" timestamp level msg

(* Calculate hash *)
let hash_content content =
  Digestif.SHA256.(to_hex (digest_string content))

(* Get cache file path *)
let cache_path key =
  let hash = hash_content key in
  let prefix = String.sub hash 0 2 in
  Filename.concat cache_dir (Filename.concat prefix hash)

(* Ensure directory exists *)
let ensure_dir path =
  let rec mkdir_p path =
    if not (Sys.file_exists path) then begin
      mkdir_p (Filename.dirname path);
      try Unix.mkdir path 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ()
    end
  in
  mkdir_p path

(* Put value in cache *)
let cache_put key value ttl =
  let path = cache_path key in
  ensure_dir (Filename.dirname path);
  
  let hash = hash_content value in
  let size = Int64.of_int (String.length value) in
  let now = Unix.gettimeofday () in
  
  (* Write to file *)
  let oc = open_out_bin path in
  output_string oc value;
  close_out oc;
  
  (* Update entry *)
  let entry = {
    key;
    hash;
    size;
    created_at = now;
    accessed_at = now;
    ttl;
  } in
  Hashtbl.replace entries key entry;
  
  log "DEBUG" (Printf.sprintf "PUT %s (%Ld bytes)" key size);
  Lwt.return_ok entry

(* Get value from cache *)
let cache_get key =
  match Hashtbl.find_opt entries key with
  | None ->
    miss_count := Int64.add !miss_count 1L;
    log "DEBUG" (Printf.sprintf "MISS %s" key);
    Lwt.return_none
  | Some entry ->
    (* Check TTL *)
    let now = Unix.gettimeofday () in
    if entry.ttl > 0L && now -. entry.created_at > Int64.to_float entry.ttl then begin
      (* Expired *)
      Hashtbl.remove entries key;
      miss_count := Int64.add !miss_count 1L;
      log "DEBUG" (Printf.sprintf "EXPIRED %s" key);
      Lwt.return_none
    end else begin
      let path = cache_path key in
      if Sys.file_exists path then begin
        let ic = open_in_bin path in
        let len = in_channel_length ic in
        let content = really_input_string ic len in
        close_in ic;
        
        (* Update access time *)
        let updated = { entry with accessed_at = now } in
        Hashtbl.replace entries key updated;
        
        hit_count := Int64.add !hit_count 1L;
        log "DEBUG" (Printf.sprintf "HIT %s" key);
        Lwt.return_some content
      end else begin
        Hashtbl.remove entries key;
        miss_count := Int64.add !miss_count 1L;
        Lwt.return_none
      end
    end

(* Delete from cache *)
let cache_delete key =
  let path = cache_path key in
  Hashtbl.remove entries key;
  if Sys.file_exists path then
    Unix.unlink path;
  log "DEBUG" (Printf.sprintf "DELETE %s" key);
  Lwt.return_ok ()

(* Garbage collection *)
let cache_gc () =
  log "INFO" "Running garbage collection...";
  let now = Unix.gettimeofday () in
  let to_remove = ref [] in
  
  Hashtbl.iter (fun key entry ->
    if entry.ttl > 0L && now -. entry.created_at > Int64.to_float entry.ttl then
      to_remove := key :: !to_remove
  ) entries;
  
  List.iter (fun key ->
    let _ = cache_delete key in ()
  ) !to_remove;
  
  log "INFO" (Printf.sprintf "GC complete: removed %d entries" (List.length !to_remove));
  Lwt.return (List.length !to_remove)

(* Get cache stats *)
let cache_stats () =
  let total_size = Hashtbl.fold (fun _ entry acc ->
    Int64.add acc entry.size
  ) entries 0L in
  {
    total_entries = Hashtbl.length entries;
    total_size;
    hit_count = !hit_count;
    miss_count = !miss_count;
  }

(* Handle cache request *)
let handle_request req =
  match req.action with
  | "get" ->
    let%lwt value = cache_get req.key in
    Lwt.return {
      success = true;
      error = None;
      value;
      hit = Option.is_some value;
      stats = None;
    }
    
  | "put" ->
    begin match req.value with
    | Some value ->
      let%lwt result = cache_put req.key value req.ttl in
      begin match result with
      | Ok _ -> Lwt.return { success = true; error = None; value = None; hit = false; stats = None }
      | Error e -> Lwt.return { success = false; error = Some e; value = None; hit = false; stats = None }
      end
    | None ->
      Lwt.return { success = false; error = Some "No value provided"; value = None; hit = false; stats = None }
    end
    
  | "delete" ->
    let%lwt _ = cache_delete req.key in
    Lwt.return { success = true; error = None; value = None; hit = false; stats = None }
    
  | "gc" ->
    let%lwt _ = cache_gc () in
    Lwt.return { success = true; error = None; value = None; hit = false; stats = Some (cache_stats ()) }
    
  | "stats" ->
    Lwt.return { success = true; error = None; value = None; hit = false; stats = Some (cache_stats ()) }
    
  | action ->
    Lwt.return { success = false; error = Some ("Unknown action: " ^ action); value = None; hit = false; stats = None }

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
    source = "cache";
    target = "broker";
    method_name = "register";
    payload = {|{"service": "cache", "type": "core"}|};
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
    | Some msg when msg.target = "cache" ->
      begin match msg.method_name with
      | "cache" ->
        begin match cache_request_of_yojson (Yojson.Safe.from_string msg.payload) with
        | Ok req ->
          let%lwt response = handle_request req in
          let response_msg = {
            version = 1;
            msg_type = Response;
            msg_id = msg.msg_id;
            source = "cache";
            target = msg.source;
            method_name = "cache";
            payload = Yojson.Safe.to_string (cache_response_to_yojson response);
            timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
            ipc_error = None;
          } in
          write_message socket response_msg
        | Error e ->
          log "ERROR" (Printf.sprintf "Invalid cache request: %s" e);
          Lwt.return_unit
        end
      | _ ->
        log "WARN" (Printf.sprintf "Unknown method: %s" msg.method_name);
        Lwt.return_unit
      end >>= loop
    | Some _ -> loop ()
  in
  loop ()

(* Periodic GC *)
let gc_loop () =
  let rec loop () =
    Lwt_unix.sleep 3600.0 >>= fun () -> (* Every hour *)
    let%lwt _ = cache_gc () in
    loop ()
  in
  loop ()

(* Main *)
let main () =
  log "INFO" "MixOS Artifact Cache starting...";
  
  ensure_dir cache_dir;
  
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
  log "INFO" "Artifact Cache ready";
  
  (* Start GC loop in background *)
  Lwt.async gc_loop;
  
  message_loop socket

let () = Lwt_main.run (main ())
