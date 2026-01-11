(* MixOS IPC Broker Service
   Central message routing for all MixOS services *)

open Lwt.Infix

(* Message types *)
type message_type = 
  | Request 
  | Response 
  | Event 
  | Stream
[@@deriving yojson]

type ipc_message = {
  version: int;
  msg_type: message_type;
  msg_id: int64;
  source: string;
  target: string;
  method_name: string; [@key "method"]
  payload: string;
  timestamp: int64;
  error: string option;
} [@@deriving yojson]

(* Client connection *)
type client = {
  id: string;
  service_name: string option;
  socket: Lwt_unix.file_descr;
  mutable subscriptions: string list;
}

(* Broker state *)
type broker_state = {
  mutable clients: client list;
  mutable msg_counter: int64;
  socket_path: string;
}

let state = {
  clients = [];
  msg_counter = 0L;
  socket_path = "/run/mixos/ipc.sock";
}

(* Logging *)
let log level msg =
  let timestamp = Unix.gettimeofday () in
  Printf.printf "[%.3f] [%s] %s\n%!" timestamp level msg

(* Read length-prefixed message *)
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

(* Write length-prefixed message *)
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

(* Find client by service name *)
let find_client_by_service name =
  List.find_opt (fun c -> c.service_name = Some name) state.clients

(* Generate message ID *)
let next_msg_id () =
  state.msg_counter <- Int64.add state.msg_counter 1L;
  state.msg_counter

(* Handle registration *)
let handle_register client msg =
  let service_name = msg.source in
  log "INFO" (Printf.sprintf "Service registered: %s" service_name);
  let updated_client = { client with service_name = Some service_name } in
  state.clients <- List.map (fun c -> 
    if c.id = client.id then updated_client else c
  ) state.clients;
  let response = {
    version = 1;
    msg_type = Response;
    msg_id = msg.msg_id;
    source = "broker";
    target = service_name;
    method_name = "register";
    payload = {|{"status": "ok"}|};
    timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
    error = None;
  } in
  write_message client.socket response

(* Route message to target *)
let route_message client msg =
  match find_client_by_service msg.target with
  | Some target_client ->
    log "DEBUG" (Printf.sprintf "Routing %s -> %s: %s" 
      msg.source msg.target msg.method_name);
    write_message target_client.socket msg
  | None ->
    log "WARN" (Printf.sprintf "Target not found: %s" msg.target);
    let error_response = {
      version = 1;
      msg_type = Response;
      msg_id = msg.msg_id;
      source = "broker";
      target = msg.source;
      method_name = msg.method_name;
      payload = "";
      timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
      error = Some (Printf.sprintf "Service not found: %s" msg.target);
    } in
    write_message client.socket error_response

(* Broadcast event to all subscribers *)
let broadcast_event event_name payload =
  let msg = {
    version = 1;
    msg_type = Event;
    msg_id = next_msg_id ();
    source = "broker";
    target = "*";
    method_name = event_name;
    payload = payload;
    timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
    error = None;
  } in
  Lwt_list.iter_p (fun client ->
    if List.mem event_name client.subscriptions || List.mem "*" client.subscriptions then
      write_message client.socket msg
    else
      Lwt.return_unit
  ) state.clients

(* Handle client connection *)
let handle_client client =
  let rec loop () =
    read_message client.socket >>= function
    | None ->
      log "INFO" (Printf.sprintf "Client disconnected: %s" client.id);
      state.clients <- List.filter (fun c -> c.id <> client.id) state.clients;
      Lwt_unix.close client.socket
    | Some msg ->
      begin match msg.method_name with
      | "register" -> handle_register client msg
      | "subscribe" ->
        let topics = String.split_on_char ',' msg.payload in
        let updated = { client with subscriptions = topics @ client.subscriptions } in
        state.clients <- List.map (fun c -> 
          if c.id = client.id then updated else c
        ) state.clients;
        Lwt.return_unit
      | _ -> route_message client msg
      end >>= loop
  in
  loop ()

(* Accept connections *)
let accept_loop server_socket =
  let rec loop () =
    Lwt_unix.accept server_socket >>= fun (client_socket, _addr) ->
    let client_id = Printf.sprintf "client_%Ld" (next_msg_id ()) in
    log "INFO" (Printf.sprintf "New connection: %s" client_id);
    let client = {
      id = client_id;
      service_name = None;
      socket = client_socket;
      subscriptions = [];
    } in
    state.clients <- client :: state.clients;
    Lwt.async (fun () -> handle_client client);
    loop ()
  in
  loop ()

(* Main *)
let main () =
  log "INFO" "MixOS Broker starting...";
  
  (* Create socket directory *)
  let socket_dir = Filename.dirname state.socket_path in
  (try Unix.mkdir socket_dir 0o755 with Unix.Unix_error (Unix.EEXIST, _, _) -> ());
  
  (* Remove existing socket *)
  (try Unix.unlink state.socket_path with Unix.Unix_error _ -> ());
  
  (* Create server socket *)
  let server_socket = Lwt_unix.socket Unix.PF_UNIX Unix.SOCK_STREAM 0 in
  let addr = Unix.ADDR_UNIX state.socket_path in
  Lwt_unix.bind server_socket addr >>= fun () ->
  Lwt_unix.listen server_socket 10;
  Unix.chmod state.socket_path 0o660;
  
  log "INFO" (Printf.sprintf "Listening on %s" state.socket_path);
  
  (* Broadcast startup event *)
  let _ = broadcast_event "broker.started" {|{"version": "0.1.0"}|} in
  
  accept_loop server_socket

let () = Lwt_main.run (main ())
