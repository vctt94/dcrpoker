// ignore_for_file: constant_identifier_names

import 'dart:async';
import 'dart:convert';

import 'package:flutter/cupertino.dart';
import 'package:json_annotation/json_annotation.dart';

part 'definitions.g.dart';

/// -------------------- JSON helpers --------------------
/// Some platforms or older backends may double-encode JSON payloads or
/// sometimes return a single object instead of a list. These helpers
/// normalize responses so call sites can be strict about types.

// Decodes a JSON string if [value] is a String. If decoding fails,
// the original value is returned. Performs up to two decode passes
// to handle possible double-encoding.
dynamic _decodeIfString(dynamic value) {
  dynamic v = value;
  for (var i = 0; i < 2 && v is String; i++) {
    try {
      v = jsonDecode(v as String);
    } catch (_) {
      break;
    }
  }
  return v;
}

// Ensures a Map<String, dynamic> by decoding if needed and throwing a
// StateError if the final value is not a Map.
Map<String, dynamic> _asJsonMap(dynamic value) {
  final v = _decodeIfString(value);
  if (v is Map) return Map<String, dynamic>.from(v as Map);
  throw StateError('Expected Map, got ${v.runtimeType}: $v');
}

// Ensures a List by decoding if needed. If the final value is a Map,
// wraps it in a single-element list. Throws if neither List nor Map.
List<dynamic> _asJsonListOrWrap(dynamic value) {
  final v = _decodeIfString(value);
  if (v is List) return v;
  if (v is Map) return [v];
  throw StateError('Expected List/Map, got ${v.runtimeType}: $v');
}

/// -------------------- Init / Identity --------------------

@JsonSerializable(explicitToJson: true)
class InitClient {
  @JsonKey(name: 'server_addr')
  final String serverAddr;
  @JsonKey(name: 'grpc_cert_path')
  final String grpcCertPath;
  @JsonKey(name: 'datadir')
  final String dataDir;
  @JsonKey(name: 'payout_address')
  final String payoutAddress;
  @JsonKey(name: 'log_file')
  final String logFile;
  @JsonKey(name: 'debug_level')
  final String debugLevel;
  @JsonKey(name: 'wants_log_ntfns')
  final bool wantsLogNtfns;

  // RPC fields
  @JsonKey(name: 'rpc_websocket_url')
  final String rpcWebsockeURL;
  @JsonKey(name: 'rpc_cert_path')
  final String rpcCertPath;
  @JsonKey(name: 'rpc_client_cert_path')
  final String rpcClientCertpath;
  @JsonKey(name: 'rpc_client_key_path')
  final String rpcClientKeypath;
  @JsonKey(name: 'rpc_user')
  final String rpcUser;
  @JsonKey(name: 'rpc_pass')
  final String rpcPass;

  InitClient(
    this.serverAddr,
    this.grpcCertPath,
    this.dataDir,
    this.payoutAddress,
    this.logFile,
    this.debugLevel,
    this.wantsLogNtfns,
    this.rpcWebsockeURL,
    this.rpcCertPath,
    this.rpcClientCertpath,
    this.rpcClientKeypath,
    this.rpcUser,
    this.rpcPass,
  );

  factory InitClient.fromJson(Map<String, dynamic> json) =>
      _$InitClientFromJson(json);
  Map<String, dynamic> toJson() => _$InitClientToJson(this);
}

@JsonSerializable(explicitToJson: true)
class InitPokerClient {
  @JsonKey(name: 'datadir')
  final String dataDir;
  @JsonKey(name: 'grpc_host')
  final String grpcHost;
  @JsonKey(name: 'grpc_port')
  final String grpcPort;
  @JsonKey(name: 'grpc_server_cert')
  final String grpcServerCert;
  @JsonKey(name: 'insecure')
  final bool insecure;
  @JsonKey(name: 'offline')
  final bool offline;
  @JsonKey(name: 'player_id')
  final String? playerId;
  @JsonKey(name: 'log_file')
  final String logFile;
  @JsonKey(name: 'debug_level')
  final String debugLevel;

  InitPokerClient(
    this.dataDir,
    this.grpcHost,
    this.grpcPort,
    this.grpcServerCert,
    this.insecure,
    this.offline,
    this.playerId,
    this.logFile,
    this.debugLevel,
  );

  factory InitPokerClient.fromJson(Map<String, dynamic> json) =>
      _$InitPokerClientFromJson(json);
  Map<String, dynamic> toJson() => _$InitPokerClientToJson(this);
}

@JsonSerializable(explicitToJson: true)
class CreateDefaultConfig {
  @JsonKey(name: 'datadir')
  final String dataDir;
  @JsonKey(name: 'server_addr')
  final String serverAddr;
  @JsonKey(name: 'grpc_cert_path')
  final String grpcCertPath;
  @JsonKey(name: 'debug_level')
  final String debugLevel;
  @JsonKey(name: 'br_rpc_url')
  final String brRpcUrl;
  @JsonKey(name: 'br_client_cert')
  final String brClientCert;
  @JsonKey(name: 'br_client_rpc_cert')
  final String brClientRpcCert;
  @JsonKey(name: 'br_client_rpc_key')
  final String brClientRpcKey;
  @JsonKey(name: 'rpc_user')
  final String rpcUser;
  @JsonKey(name: 'rpc_pass')
  final String rpcPass;

  CreateDefaultConfig(
    this.dataDir,
    this.serverAddr,
    this.grpcCertPath,
    this.debugLevel,
    this.brRpcUrl,
    this.brClientCert,
    this.brClientRpcCert,
    this.brClientRpcKey,
    this.rpcUser,
    this.rpcPass,
  );

  factory CreateDefaultConfig.fromJson(Map<String, dynamic> json) =>
      _$CreateDefaultConfigFromJson(json);
  Map<String, dynamic> toJson() => _$CreateDefaultConfigToJson(this);
}

@JsonSerializable()
class IDInit {
  @JsonKey(name: 'id')
  final String uid;
  @JsonKey(name: 'nick')
  final String nick;
  IDInit(this.uid, this.nick);

  factory IDInit.fromJson(Map<String, dynamic> json) => _$IDInitFromJson(json);
  Map<String, dynamic> toJson() => _$IDInitToJson(this);
}

@JsonSerializable()
class GetUserNickArgs {
  @JsonKey(name: 'uid')
  final String uid;

  GetUserNickArgs(this.uid);
  factory GetUserNickArgs.fromJson(Map<String, dynamic> json) =>
      _$GetUserNickArgsFromJson(json);
  Map<String, dynamic> toJson() => _$GetUserNickArgsToJson(this);
}

/// -------------------- Local types (WR shim) --------------------
/// Keep these proto-agnostic; adapt in a separate file if needed.

@JsonSerializable()
class LocalPlayer {
  @JsonKey(name: 'uid')
  final String uid;
  @JsonKey(name: 'nick')
  final String? nick;
  @JsonKey(name: 'bet_amt')
  final int betAmount;
  @JsonKey(name: 'ready')
  final bool ready;

  LocalPlayer(
    this.uid,
    this.nick,
    this.betAmount, {
    this.ready = false,
  });

  factory LocalPlayer.fromJson(Map<String, dynamic> json) =>
      _$LocalPlayerFromJson(json);
  Map<String, dynamic> toJson() => _$LocalPlayerToJson(this);
}

@JsonSerializable(explicitToJson: true)
class LocalWaitingRoom {
  @JsonKey(name: 'id')
  final String id;
  @JsonKey(name: 'host_id')
  final String host;
  @JsonKey(name: 'bet_amt')
  final int betAmt;
  @JsonKey(name: 'players', defaultValue: [])
  final List<LocalPlayer> players;

  const LocalWaitingRoom(
    this.id,
    this.host,
    this.betAmt, {
    this.players = const [],
  });

  factory LocalWaitingRoom.fromJson(Map<String, dynamic> json) =>
      _$LocalWaitingRoomFromJson(json);
  Map<String, dynamic> toJson() => _$LocalWaitingRoomToJson(this);
}

@JsonSerializable()
class LocalInfo {
  final String id;
  final String nick;
  LocalInfo(this.id, this.nick);
  factory LocalInfo.fromJson(Map<String, dynamic> json) =>
      _$LocalInfoFromJson(json);
  Map<String, dynamic> toJson() => _$LocalInfoToJson(this);
}

@JsonSerializable()
class ServerCert {
  @JsonKey(name: "inner_fingerprint")
  final String innerFingerprint;
  @JsonKey(name: "outer_fingerprint")
  final String outerFingerprint;
  const ServerCert(this.innerFingerprint, this.outerFingerprint);

  factory ServerCert.fromJson(Map<String, dynamic> json) =>
      _$ServerCertFromJson(json);
  Map<String, dynamic> toJson() => _$ServerCertToJson(this);
}

const connStateOffline = 0;
const connStateCheckingWallet = 1;
const connStateOnline = 2;

@JsonSerializable()
class ServerInfo {
  final String innerFingerprint;
  final String outerFingerprint;
  final String serverAddr;
  const ServerInfo({
    required this.innerFingerprint,
    required this.outerFingerprint,
    required this.serverAddr,
  });
  const ServerInfo.empty()
      : this(innerFingerprint: "", outerFingerprint: "", serverAddr: "");

  factory ServerInfo.fromJson(Map<String, dynamic> json) =>
      _$ServerInfoFromJson(json);
  Map<String, dynamic> toJson() => _$ServerInfoToJson(this);
}

@JsonSerializable()
class RemoteUser {
  final String uid;
  final String nick;

  const RemoteUser(this.uid, this.nick);

  factory RemoteUser.fromJson(Map<String, dynamic> json) =>
      _$RemoteUserFromJson(json);
  Map<String, dynamic> toJson() => _$RemoteUserToJson(this);
}

@JsonSerializable()
class PublicIdentity {
  final String name;
  final String nick;
  final String identity;

  PublicIdentity(this.name, this.nick, this.identity);
  factory PublicIdentity.fromJson(Map<String, dynamic> json) =>
      _$PublicIdentityFromJson(json);
  Map<String, dynamic> toJson() => _$PublicIdentityToJson(this);
}

@JsonSerializable()
class Account {
  final String name;
  @JsonKey(name: "unconfirmed_balance")
  final int unconfirmedBalance;
  @JsonKey(name: "confirmed_balance")
  final int confirmedBalance;
  @JsonKey(name: "internal_key_count")
  final int internalKeyCount;
  @JsonKey(name: "external_key_count")
  final int externalKeyCount;

  Account(this.name, this.unconfirmedBalance, this.confirmedBalance,
      this.internalKeyCount, this.externalKeyCount);

  factory Account.fromJson(Map<String, dynamic> json) =>
      _$AccountFromJson(json);
  Map<String, dynamic> toJson() => _$AccountToJson(this);
}

@JsonSerializable()
class LogEntry {
  final String from;
  final String message;
  final bool internal;
  final int timestamp;
  LogEntry(this.from, this.message, this.internal, this.timestamp);

  factory LogEntry.fromJson(Map<String, dynamic> json) =>
      _$LogEntryFromJson(json);
  Map<String, dynamic> toJson() => _$LogEntryToJson(this);
}

@JsonSerializable()
class SendOnChain {
  final String addr;
  final int amount;
  @JsonKey(name: "from_account")
  final String fromAccount;

  SendOnChain(this.addr, this.amount, this.fromAccount);
  Map<String, dynamic> toJson() => _$SendOnChainToJson(this);
}

@JsonSerializable()
class LoadUserHistory {
  final String uid;
  @JsonKey(name: "is_gc")
  final bool isGC;
  final int page;
  @JsonKey(name: "page_num")
  final int pageNum;

  LoadUserHistory(this.uid, this.isGC, this.page, this.pageNum);
  Map<String, dynamic> toJson() => _$LoadUserHistoryToJson(this);
}

@JsonSerializable()
class WriteInvite {
  @JsonKey(name: "fund_amount")
  final int fundAmount;
  @JsonKey(name: "fund_account")
  final String fundAccount;
  @JsonKey(name: "gc_id")
  final String? gcid;
  final bool prepaid;

  WriteInvite(this.fundAmount, this.fundAccount, this.gcid, this.prepaid);
  Map<String, dynamic> toJson() => _$WriteInviteToJson(this);
}

@JsonSerializable()
class RedeemedInviteFunds {
  final String txid;
  final int total;

  RedeemedInviteFunds(this.txid, this.total);
  factory RedeemedInviteFunds.fromJson(Map<String, dynamic> json) =>
      _$RedeemedInviteFundsFromJson(json);
  Map<String, dynamic> toJson() => _$RedeemedInviteFundsToJson(this);
}

@JsonSerializable()
class CreateWaitingRoomArgs {
  @JsonKey(name: 'client_id')
  final String clientId;
  @JsonKey(name: 'bet_amt')
  final int betAmt;
  @JsonKey(name: 'escrow_id')
  final String? escrowId;

  CreateWaitingRoomArgs(this.clientId, this.betAmt, {this.escrowId});

  factory CreateWaitingRoomArgs.fromJson(Map<String, dynamic> json) =>
      _$CreateWaitingRoomArgsFromJson(json);

  Map<String, dynamic> toJson() => _$CreateWaitingRoomArgsToJson(this);
}

@JsonSerializable()
class PokerTable {
  @JsonKey(name: 'id')
  final String id;
  @JsonKey(name: 'host_id')
  final String hostId;
  @JsonKey(name: 'small_blind')
  final int smallBlind;
  @JsonKey(name: 'big_blind')
  final int bigBlind;
  @JsonKey(name: 'max_players')
  final int maxPlayers;
  @JsonKey(name: 'min_players')
  final int minPlayers;
  @JsonKey(name: 'current_players')
  final int currentPlayers;
  @JsonKey(name: 'min_balance')
  final int minBalance;
  @JsonKey(name: 'buy_in')
  final int buyIn;
  @JsonKey(name: 'game_started')
  final bool gameStarted;
  @JsonKey(name: 'all_players_ready')
  final bool allPlayersReady;

  PokerTable(
    this.id,
    this.hostId,
    this.smallBlind,
    this.bigBlind,
    this.maxPlayers,
    this.minPlayers,
    this.currentPlayers,
    this.minBalance,
    this.buyIn,
    this.gameStarted,
    this.allPlayersReady,
  );

  factory PokerTable.fromJson(Map<String, dynamic> json) =>
      _$PokerTableFromJson(json);
  Map<String, dynamic> toJson() => _$PokerTableToJson(this);
}

@JsonSerializable()
class CreatePokerTableArgs {
  @JsonKey(name: 'small_blind')
  final int smallBlind;
  @JsonKey(name: 'big_blind')
  final int bigBlind;
  @JsonKey(name: 'max_players')
  final int maxPlayers;
  @JsonKey(name: 'min_players')
  final int minPlayers;
  @JsonKey(name: 'min_balance')
  final int minBalance;
  @JsonKey(name: 'buy_in')
  final int buyIn;
  @JsonKey(name: 'starting_chips')
  final int startingChips;
  @JsonKey(name: 'time_bank_seconds')
  final int timeBankSeconds;
  @JsonKey(name: 'auto_start_ms')
  final int autoStartMs;

  CreatePokerTableArgs(
    this.smallBlind,
    this.bigBlind,
    this.maxPlayers,
    this.minPlayers,
    this.minBalance,
    this.buyIn,
    this.startingChips,
    this.timeBankSeconds,
    this.autoStartMs,
  );

  factory CreatePokerTableArgs.fromJson(Map<String, dynamic> json) =>
      _$CreatePokerTableArgsFromJson(json);
  Map<String, dynamic> toJson() => _$CreatePokerTableArgsToJson(this);
}

@JsonSerializable()
class JoinPokerTableArgs {
  @JsonKey(name: 'table_id')
  final String tableId;

  JoinPokerTableArgs(this.tableId);

  factory JoinPokerTableArgs.fromJson(Map<String, dynamic> json) =>
      _$JoinPokerTableArgsFromJson(json);
  Map<String, dynamic> toJson() => _$JoinPokerTableArgsToJson(this);
}

@JsonSerializable()
class RunState {
  @JsonKey(name: "dcrlnd_running")
  final bool dcrlndRunning;
  @JsonKey(name: "client_running")
  final bool clientRunning;

  RunState({required this.dcrlndRunning, required this.clientRunning});
  factory RunState.fromJson(Map<String, dynamic> json) =>
      _$RunStateFromJson(json);
  Map<String, dynamic> toJson() => _$RunStateToJson(this);
}

@JsonSerializable()
class ZipLogsArgs {
  @JsonKey(name: "include_golib")
  final bool includeGolib;
  @JsonKey(name: "include_ln")
  final bool includeLn;
  @JsonKey(name: "only_last_file")
  final bool onlyLastFile;
  @JsonKey(name: "dest_path")
  final String destPath;

  ZipLogsArgs(this.includeGolib, this.includeLn, this.onlyLastFile, this.destPath);
  Map<String, dynamic> toJson() => _$ZipLogsArgsToJson(this);
}

/// -------------------- UI Notifications --------------------

const String UINtfnPM = "pm";
const String UINtfnGCM = "gcm";
const String UINtfnGCMMention = "gcmmention";
const String UINtfnMultiple = "multiple";

@JsonSerializable()
class UINotification {
  final String type;
  final String text;
  final int count;
  final String from;

  UINotification(this.type, this.text, this.count, this.from);
  factory UINotification.fromJson(Map<String, dynamic> json) =>
      _$UINotificationFromJson(json);
  Map<String, dynamic> toJson() => _$UINotificationToJson(this);
}

@JsonSerializable()
class UINotificationsConfig {
  final bool pms;
  final bool gcms;
  @JsonKey(name: "gcmentions")
  final bool gcMentions;

  UINotificationsConfig(this.pms, this.gcms, this.gcMentions);
  factory UINotificationsConfig.disabled() =>
      UINotificationsConfig(false, false, false);
  factory UINotificationsConfig.fromJson(Map<String, dynamic> json) =>
      _$UINotificationsConfigFromJson(json);
  Map<String, dynamic> toJson() => _$UINotificationsConfigToJson(this);
}

/// -------------------- Notifications mixin --------------------

mixin NtfStreams {
  final StreamController<RemoteUser> ntfAcceptedInvites =
      StreamController<RemoteUser>.broadcast();
  Stream<RemoteUser> acceptedInvites() => ntfAcceptedInvites.stream;

  final StreamController<String> ntfLogLines =
      StreamController<String>.broadcast();
  Stream<String> logLines() => ntfLogLines.stream;

  final StreamController<int> ntfRescanProgress =
      StreamController<int>.broadcast();
  Stream<int> rescanWalletProgress() => ntfRescanProgress.stream;

  final StreamController<UINotification> ntfUINotifications =
      StreamController<UINotification>.broadcast();
  Stream<UINotification> uiNotifications() => ntfUINotifications.stream;

  // Waiting room notifications
  final StreamController<LocalWaitingRoom> ntfWaitingRoomCreated =
      StreamController<LocalWaitingRoom>.broadcast();
  Stream<LocalWaitingRoom> waitingRoomCreated() => ntfWaitingRoomCreated.stream;

  void disposeNtfStreams() {
    ntfAcceptedInvites.close();
    ntfLogLines.close();
    ntfRescanProgress.close();
    ntfUINotifications.close();
    ntfWaitingRoomCreated.close();
  }

  void handleNotifications(int cmd, bool isError, String jsonPayload) {
    // If you need payload, parse it here:
    // final data = jsonPayload.isNotEmpty ? jsonDecode(jsonPayload) : null;

    switch (cmd) {
      case NTNOP:
        break;

      case NTWRCreated:
        try {
          if (jsonPayload.isNotEmpty) {
            final data = jsonDecode(jsonPayload);
            final wr = LocalWaitingRoom.fromJson(Map<String, dynamic>.from(data as Map));
            ntfWaitingRoomCreated.add(wr);
          }
        } catch (e) {
          debugPrint("Failed to parse NTWRCreated payload: $e");
        }
        break;
      default:
        debugPrint("Received unknown notification ${cmd.toRadixString(16)}");
    }
  }
}

/// -------------------- Platform bridge --------------------

abstract class PluginPlatform {
  Future<String?> get platformVersion => Future.error("unimplemented");
  String get majorPlatform => "unknown-major-plat";
  String get minorPlatform => "unknown-minor-plat";

  Future<void> setTag(String tag) async => Future.error("unimplemented");
  Future<void> hello() async => Future.error("unimplemented");
  Future<String> getURL(String url) async => Future.error("unimplemented");
  Future<String> nextTime() async => Future.error("unimplemented");
  Future<void> writeStr(String s) async => Future.error("unimplemented");
  Stream<String> readStream() async* {
    throw "unimplemented";
  }

  // Android only (no-ops elsewhere)
  Future<void> startForegroundSvc() => Future.error("unimplemented");
  Future<void> stopForegroundSvc() => Future.error("unimplemented");
  Future<void> setNtfnsEnabled(bool enabled) => Future.error("unimplemented");

  Future<dynamic> asyncCall(int cmd, dynamic payload) async =>
      Future.error("unimplemented");

  // Notification streams (must be provided by platform mixins)
  Stream<LocalWaitingRoom> waitingRoomCreated();

  Future<String> asyncHello(String name) async {
    final r = await asyncCall(CTHello, name);
    return r as String;
    // If platform returns non-string, this will throw; that’s desirable.
  }

  Future<LocalInfo> initClient(InitClient args) async {
    final res = await asyncCall(CTInitClient, args.toJson());
    print("DEBUG: InitClient response: $res");
    return LocalInfo.fromJson(_asJsonMap(res));
  }


  Future<Map<String, dynamic>> createDefaultConfig(CreateDefaultConfig args) async {
    final res = await asyncCall(CTCreateDefaultConfig, args.toJson());
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> createDefaultServerCert(String certPath) async {
    final res = await asyncCall(CTCreateDefaultServerCert, certPath);
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> loadConfig(String filepath) async {
    final res = await asyncCall(CTLoadConfig, filepath);
    return _asJsonMap(res);
  }

  Future<void> createLockFile(String rootDir) async =>
      await asyncCall(CTCreateLockFile, rootDir);
  Future<void> closeLockFile(String rootDir) async =>
      await asyncCall(CTCloseLockFile, rootDir);
  Future<String> userNick(String pid) async {
    final r = await asyncCall(CTGetUserNick, pid);
    return r as String;
  }

  Future<List<LocalPlayer>> getWRPlayers() async {
    final res = await asyncCall(CTGetWRPlayers, "");
    if (res == null) return [];
    final list = _asJsonListOrWrap(res);
    return list.map((v) {
      final item = _decodeIfString(v);
      return LocalPlayer.fromJson(Map<String, dynamic>.from(item as Map));
    }).toList();
  }

  Future<List<LocalWaitingRoom>> getWaitingRooms() async {
    final res = await asyncCall(CTGetWaitingRooms, "");
    if (res == null) return [];
    final list = _asJsonListOrWrap(res);
    return list.map((v) {
      final item = _decodeIfString(v);
      return LocalWaitingRoom.fromJson(Map<String, dynamic>.from(item as Map));
    }).toList();
  }

  Future<LocalWaitingRoom> JoinWaitingRoom(String id, {String? escrowId}) async {
    final payload = <String, dynamic>{
      'room_id': id,
      'escrow_id': escrowId ?? '',
    };
    final response = await asyncCall(CTJoinWaitingRoom, payload);
    final resp = _decodeIfString(response);
    if (resp is Map) {
      return LocalWaitingRoom.fromJson(Map<String, dynamic>.from(resp));
    }
    throw Exception("Invalid JoinWaitingRoom response: $response");
  }

  Future<LocalWaitingRoom> CreateWaitingRoom(CreateWaitingRoomArgs args) async {
    // Always send `escrow_id` key (even if empty) for stable decoding
    final payload = <String, dynamic>{
      'client_id': args.clientId,
      'bet_amt': args.betAmt,
      'escrow_id': args.escrowId ?? '',
    };
    final response = await asyncCall(CTCreateWaitingRoom, payload);
    final resp = _decodeIfString(response);
    if (resp is Map) {
      return LocalWaitingRoom.fromJson(Map<String, dynamic>.from(resp));
    }
    throw Exception("Invalid CreateWaitingRoom response: $response");
  }

  Future<void> LeaveWaitingRoom(String id) async {
    await asyncCall(CTLeaveWaitingRoom, id);
  }

  // Escrow/Settlement methods
  Future<Map<String, String>> generateSettlementSessionKey() async {
    final res = await asyncCall(CTGenerateSessionKey, "");
    final m = _asJsonMap(res);
    return m.map((k, v) => MapEntry(k, v == null ? '' : v.toString()));
  }

  Future<Map<String, dynamic>> openEscrow({
    required String payout,
    required int betAtoms,
    int csvBlocks = 64,
  }) async {
    final payload = {
      'payout': payout,
      'bet_atoms': betAtoms,
      'csv_blocks': csvBlocks,
    };
    final res = await asyncCall(CTOpenEscrow, payload);
    return _asJsonMap(res);
  }

  Future<void> startPreSign(String matchId) async {
    await asyncCall(CTStartPreSign, {'match_id': matchId});
  }

  Future<void> archiveSettlementSessionKey(String matchId) async {
    await asyncCall(CTArchiveSessionKey, {'match_id': matchId});
  }

  // Poker table methods
  Future<List<PokerTable>> getPokerTables() async {
    final res = await asyncCall(CTGetPokerTables, "");
    if (res == null) return [];

    final list = _asJsonListOrWrap(res);
    print("DEBUG: GetPokerTables response: $list");

    return list.map((v) {
      final item = _decodeIfString(v);
      if (item is! Map) {
        throw Exception("Invalid table item type: ${item.runtimeType}");
      }
      // Go DTO now guarantees all fields are properly typed
      return PokerTable.fromJson(Map<String, dynamic>.from(item));
    }).toList();
  }

  Future<Map<String, dynamic>> joinPokerTable(JoinPokerTableArgs args) async {
    final res = await asyncCall(CTJoinPokerTable, args.toJson());
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> createPokerTable(CreatePokerTableArgs args) async {
    final res = await asyncCall(CTCreatePokerTable, args.toJson());
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> leavePokerTable() async {
    final res = await asyncCall(CTLeavePokerTable, "");
    return _asJsonMap(res);
  }

  Future<Map<String, int>> getPokerBalance() async {
    final res = await asyncCall(CTGetPokerBalance, "");
    final m = _asJsonMap(res);
    return m.map((k, v) => MapEntry(k, (v as num).toInt()));
  }

  Future<String> getPokerCurrentTable() async {
    final res = await asyncCall(CTGetPlayerCurrentTable, "");
    final map = _asJsonMap(res);
    if (!map.containsKey('table_id') || map['table_id'] is! String) {
      throw StateError('CTGetPlayerCurrentTable: missing or invalid table_id');
    }
    return map['table_id'] as String;
  }
}

/// -------------------- Commands & Notifications --------------------

const int CTUnknown = 0x00;
const int CTHello = 0x01;
const int CTInitClient = 0x02;
const int CTGetUserNick = 0x03;
const int CTCreateLockFile = 0x04;
const int CTGetWRPlayers = 0x05;
const int CTGetWaitingRooms = 0x06;
const int CTJoinWaitingRoom = 0x07;
const int CTCreateWaitingRoom = 0x08;
const int CTLeaveWaitingRoom = 0x09;
const int CTGenerateSessionKey = 0x0a;
const int CTOpenEscrow        = 0x0b;
const int CTStartPreSign      = 0x0c;
// 0x0d unused
const int CTArchiveSessionKey = 0x0e;

// Poker-specific commands
const int CTGetPlayerCurrentTable = 0x10;
const int CTLoadConfig       = 0x11;
const int CTGetPokerTables    = 0x12;
const int CTJoinPokerTable    = 0x13;
const int CTCreatePokerTable  = 0x14;
const int CTLeavePokerTable   = 0x15;
const int CTGetPokerBalance   = 0x16;
const int CTCreateDefaultConfig = 0x17;
const int CTCreateDefaultServerCert = 0x18;

const int CTCloseLockFile     = 0x60;

const int notificationsStartID = 0x1000;
const int notificationClientStopped = 0x1001; // kept for compatibility; actual client-stopped is 0x1002 in golib
const int NTNOP = 0x1004;
const int NTWRCreated = 0x1005;
