(* MixOS Dependency Graph Resolver
   Topological sorting and version constraint solving *)

open Lwt.Infix

(* Version constraint types *)
type version_op = Eq | Ge | Le | Gt | Lt | Ne [@@deriving yojson]

type version_constraint = {
  op: version_op;
  version: string;
} [@@deriving yojson]

type dependency = {
  name: string;
  constraints: version_constraint list;
  optional: bool;
} [@@deriving yojson]

type package_def = {
  name: string;
  version: string;
  dependencies: dependency list;
  provides: string list;
  conflicts: string list;
} [@@deriving yojson]

(* Resolution types *)
type resolve_request = {
  packages: string list;
  include_optional: bool;
} [@@deriving yojson]

type dependency_node = {
  dep_name: string; [@key "name"]
  dep_version: string; [@key "version"]
  dep_dependencies: string list; [@key "dependencies"]
  is_installed: bool;
} [@@deriving yojson]

type resolve_response = {
  success: bool;
  error: string option;
  graph: dependency_node list;
  install_order: string list;
} [@@deriving yojson]

(* Package database *)
let package_db : (string, package_def list) Hashtbl.t = Hashtbl.create 100
let installed : (string, string) Hashtbl.t = Hashtbl.create 100

(* Logging *)
let log level msg =
  let timestamp = Unix.gettimeofday () in
  Printf.printf "[%.3f] [resolver] [%s] %s\n%!" timestamp level msg

(* Version comparison *)
let compare_versions v1 v2 =
  let parse_version v =
    String.split_on_char '.' v
    |> List.map (fun s -> try int_of_string s with _ -> 0)
  in
  let rec compare_parts p1 p2 =
    match p1, p2 with
    | [], [] -> 0
    | [], _ -> -1
    | _, [] -> 1
    | h1 :: t1, h2 :: t2 ->
      if h1 < h2 then -1
      else if h1 > h2 then 1
      else compare_parts t1 t2
  in
  compare_parts (parse_version v1) (parse_version v2)

(* Check version constraint *)
let check_constraint version constraint_ =
  let cmp = compare_versions version constraint_.version in
  match constraint_.op with
  | Eq -> cmp = 0
  | Ge -> cmp >= 0
  | Le -> cmp <= 0
  | Gt -> cmp > 0
  | Lt -> cmp < 0
  | Ne -> cmp <> 0

(* Find best matching version *)
let find_best_version name constraints =
  match Hashtbl.find_opt package_db name with
  | None -> None
  | Some versions ->
    let matching = List.filter (fun pkg ->
      List.for_all (check_constraint pkg.version) constraints
    ) versions in
    (* Return highest version *)
    match List.sort (fun a b -> compare_versions b.version a.version) matching with
    | [] -> None
    | best :: _ -> Some best

(* Build dependency graph *)
let build_graph packages include_optional =
  let visited = Hashtbl.create 100 in
  let graph = ref [] in
  
  let rec visit name constraints =
    if Hashtbl.mem visited name then ()
    else begin
      Hashtbl.add visited name true;
      match find_best_version name constraints with
      | None ->
        log "WARN" (Printf.sprintf "Package not found: %s" name)
      | Some pkg ->
        (* Visit dependencies first *)
        List.iter (fun dep ->
          if not dep.optional || include_optional then
            visit dep.name dep.constraints
        ) pkg.dependencies;
        
        (* Add to graph *)
        let node = {
          dep_name = pkg.name;
          dep_version = pkg.version;
          dep_dependencies = List.filter_map (fun d ->
            if not d.optional || include_optional then Some d.name else None
          ) pkg.dependencies;
          is_installed = Hashtbl.mem installed pkg.name;
        } in
        graph := node :: !graph
    end
  in
  
  List.iter (fun name -> visit name []) packages;
  !graph

(* Topological sort for install order *)
let topological_sort graph =
  let in_degree = Hashtbl.create 100 in
  let adj = Hashtbl.create 100 in
  
  (* Initialize *)
  List.iter (fun node ->
    Hashtbl.replace in_degree node.dep_name 0;
    Hashtbl.replace adj node.dep_name [];
  ) graph;
  
  (* Build adjacency and in-degree *)
  List.iter (fun node ->
    List.iter (fun dep ->
      if Hashtbl.mem in_degree dep then begin
        let current = Hashtbl.find adj dep in
        Hashtbl.replace adj dep (node.dep_name :: current);
        let deg = Hashtbl.find in_degree node.dep_name in
        Hashtbl.replace in_degree node.dep_name (deg + 1)
      end
    ) node.dep_dependencies
  ) graph;
  
  (* Kahn's algorithm *)
  let queue = Queue.create () in
  Hashtbl.iter (fun name deg ->
    if deg = 0 then Queue.add name queue
  ) in_degree;
  
  let result = ref [] in
  while not (Queue.is_empty queue) do
    let name = Queue.pop queue in
    result := name :: !result;
    List.iter (fun neighbor ->
      let deg = Hashtbl.find in_degree neighbor - 1 in
      Hashtbl.replace in_degree neighbor deg;
      if deg = 0 then Queue.add neighbor queue
    ) (Hashtbl.find adj name)
  done;
  
  List.rev !result

(* Detect cycles *)
let detect_cycles graph =
  let white = Hashtbl.create 100 in
  let gray = Hashtbl.create 100 in
  let black = Hashtbl.create 100 in
  
  List.iter (fun node -> Hashtbl.add white node.dep_name true) graph;
  
  let adj = Hashtbl.create 100 in
  List.iter (fun node ->
    Hashtbl.replace adj node.dep_name node.dep_dependencies
  ) graph;
  
  let rec dfs name =
    if Hashtbl.mem black name then false
    else if Hashtbl.mem gray name then true (* cycle! *)
    else begin
      Hashtbl.remove white name;
      Hashtbl.add gray name true;
      let has_cycle = List.exists dfs (
        try Hashtbl.find adj name with Not_found -> []
      ) in
      Hashtbl.remove gray name;
      Hashtbl.add black name true;
      has_cycle
    end
  in
  
  Hashtbl.fold (fun name _ acc ->
    acc || dfs name
  ) white false

(* Handle resolve request *)
let handle_resolve req =
  log "INFO" (Printf.sprintf "Resolving: %s" (String.concat ", " req.packages));
  
  let graph = build_graph req.packages req.include_optional in
  
  if detect_cycles graph then begin
    log "ERROR" "Dependency cycle detected";
    Lwt.return {
      success = false;
      error = Some "Dependency cycle detected";
      graph = [];
      install_order = [];
    }
  end else begin
    let order = topological_sort graph in
    (* Filter out already installed *)
    let to_install = List.filter (fun name ->
      not (Hashtbl.mem installed name)
    ) order in
    
    log "INFO" (Printf.sprintf "Resolution complete: %d packages" (List.length to_install));
    Lwt.return {
      success = true;
      error = None;
      graph;
      install_order = to_install;
    }
  end

(* Add package to database *)
let add_package pkg =
  let versions = match Hashtbl.find_opt package_db pkg.name with
    | Some v -> pkg :: v
    | None -> [pkg]
  in
  Hashtbl.replace package_db pkg.name versions

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
    source = "resolver";
    target = "broker";
    method_name = "register";
    payload = {|{"service": "resolver", "type": "core"}|};
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
    | Some msg when msg.target = "resolver" ->
      begin match msg.method_name with
      | "resolve" ->
        begin match resolve_request_of_yojson (Yojson.Safe.from_string msg.payload) with
        | Ok req ->
          let%lwt response = handle_resolve req in
          let response_msg = {
            version = 1;
            msg_type = Response;
            msg_id = msg.msg_id;
            source = "resolver";
            target = msg.source;
            method_name = "resolve";
            payload = Yojson.Safe.to_string (resolve_response_to_yojson response);
            timestamp = Int64.of_float (Unix.gettimeofday () *. 1000.0);
            ipc_error = None;
          } in
          write_message socket response_msg
        | Error e ->
          log "ERROR" (Printf.sprintf "Invalid resolve request: %s" e);
          Lwt.return_unit
        end
      | "add_package" ->
        begin match package_def_of_yojson (Yojson.Safe.from_string msg.payload) with
        | Ok pkg ->
          add_package pkg;
          log "INFO" (Printf.sprintf "Added package: %s %s" pkg.name pkg.version);
          Lwt.return_unit
        | Error e ->
          log "ERROR" (Printf.sprintf "Invalid package def: %s" e);
          Lwt.return_unit
        end
      | _ ->
        log "WARN" (Printf.sprintf "Unknown method: %s" msg.method_name);
        Lwt.return_unit
      end >>= loop
    | Some _ -> loop ()
  in
  loop ()

(* Initialize with some test packages *)
let init_test_packages () =
  add_package { name = "libc"; version = "1.0.0"; dependencies = []; provides = ["c-library"]; conflicts = [] };
  add_package { name = "gcc"; version = "13.0.0"; dependencies = [{ name = "libc"; constraints = []; optional = false }]; provides = []; conflicts = [] };
  add_package { name = "make"; version = "4.4.0"; dependencies = [{ name = "libc"; constraints = []; optional = false }]; provides = []; conflicts = [] };
  add_package { name = "go"; version = "1.21.0"; dependencies = [{ name = "libc"; constraints = []; optional = false }]; provides = []; conflicts = [] };
  add_package { name = "ocaml"; version = "5.1.0"; dependencies = [{ name = "libc"; constraints = []; optional = false }]; provides = []; conflicts = [] };
  add_package { name = "python"; version = "3.12.0"; dependencies = [{ name = "libc"; constraints = []; optional = false }]; provides = []; conflicts = [] }

(* Main *)
let main () =
  log "INFO" "MixOS Dependency Resolver starting...";
  
  init_test_packages ();
  
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
  log "INFO" "Dependency Resolver ready";
  message_loop socket

let () = Lwt_main.run (main ())
