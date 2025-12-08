// ignore_for_file: constant_identifier_names

import 'dart:async';
import 'dart:convert';

import 'package:flutter/cupertino.dart';
import 'package:fixnum/fixnum.dart';
import 'package:json_annotation/json_annotation.dart';
import 'grpc/generated/poker.pb.dart' as pr;

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
  @JsonKey(name: 'sounds_enabled')
  final bool soundsEnabled;

  InitClient(
    this.serverAddr,
    this.grpcCertPath,
    this.dataDir,
    this.payoutAddress,
    this.logFile,
    this.debugLevel,
    this.soundsEnabled,
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
  @JsonKey(name: 'sounds_enabled')
  final bool soundsEnabled;

  CreateDefaultConfig(
    this.dataDir,
    this.serverAddr,
    this.grpcCertPath,
    this.debugLevel,
    this.soundsEnabled,
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

  LocalPlayer(this.uid, this.nick, this.betAmount, {this.ready = false});

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
class RegisterRequest {
  final String nickname;
  RegisterRequest(this.nickname);
  factory RegisterRequest.fromJson(Map<String, dynamic> json) =>
      _$RegisterRequestFromJson(json);
  Map<String, dynamic> toJson() => _$RegisterRequestToJson(this);
}

@JsonSerializable()
class LoginRequest {
  final String nickname;
  LoginRequest(this.nickname);
  factory LoginRequest.fromJson(Map<String, dynamic> json) =>
      _$LoginRequestFromJson(json);
  Map<String, dynamic> toJson() => _$LoginRequestToJson(this);
}

@JsonSerializable()
class LoginResponse {
  @JsonKey(name: 'token')
  final String token;
  @JsonKey(name: 'user_id')
  final String userId;
  @JsonKey(name: 'nickname')
  final String nickname;
  @JsonKey(name: 'address')
  final String address;
  LoginResponse(this.token, this.userId, this.nickname, this.address);
  factory LoginResponse.fromJson(Map<String, dynamic> json) =>
      _$LoginResponseFromJson(json);
  Map<String, dynamic> toJson() => _$LoginResponseToJson(this);
}

@JsonSerializable()
class RequestLoginCodeResponse {
  final String code;
  @JsonKey(name: 'ttl_sec')
  final int ttlSec;
  @JsonKey(name: 'address_hint')
  final String addressHint;
  RequestLoginCodeResponse(this.code, this.ttlSec, this.addressHint);
  factory RequestLoginCodeResponse.fromJson(Map<String, dynamic> json) =>
      _$RequestLoginCodeResponseFromJson(json);
  Map<String, dynamic> toJson() => _$RequestLoginCodeResponseToJson(this);
}

@JsonSerializable()
class SetPayoutAddressRequest {
  final String address;
  final String signature;
  final String code;
  SetPayoutAddressRequest(this.address, this.signature, this.code);
  factory SetPayoutAddressRequest.fromJson(Map<String, dynamic> json) =>
      _$SetPayoutAddressRequestFromJson(json);
  Map<String, dynamic> toJson() => _$SetPayoutAddressRequestToJson(this);
}

@JsonSerializable()
class SetPayoutAddressResponse {
  @JsonKey(defaultValue: false)
  final bool ok;
  @JsonKey(defaultValue: '')
  final String error;
  @JsonKey(defaultValue: '')
  final String address;
  SetPayoutAddressResponse(this.ok, this.error, this.address);
  factory SetPayoutAddressResponse.fromJson(Map<String, dynamic> json) =>
      _$SetPayoutAddressResponseFromJson(json);
  Map<String, dynamic> toJson() => _$SetPayoutAddressResponseToJson(this);
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

  Account(
    this.name,
    this.unconfirmedBalance,
    this.confirmedBalance,
    this.internalKeyCount,
    this.externalKeyCount,
  );

  factory Account.fromJson(Map<String, dynamic> json) =>
      _$AccountFromJson(json);
  Map<String, dynamic> toJson() => _$AccountToJson(this);
}

enum EscrowNotificationType { escrowFunding, other }

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
  @JsonKey(name: 'buy_in')
  final int buyIn;
  @JsonKey(name: 'game_started')
  final bool gameStarted;
  @JsonKey(name: 'all_players_ready')
  final bool allPlayersReady;
  @JsonKey(name: 'players')
  final List<PlayerDTO>? players; // Optional: included in notifications

  PokerTable(
    this.id,
    this.hostId,
    this.smallBlind,
    this.bigBlind,
    this.maxPlayers,
    this.minPlayers,
    this.currentPlayers,
    this.buyIn,
    this.gameStarted,
    this.allPlayersReady, {
    this.players,
  });

  factory PokerTable.fromJson(Map<String, dynamic> json) =>
      _$PokerTableFromJson(json);
  Map<String, dynamic> toJson() => _$PokerTableToJson(this);

  pr.Table toProtobuf() {
    final t = pr.Table()
      ..id = id
      ..hostId = hostId
      ..smallBlind = Int64(smallBlind)
      ..bigBlind = Int64(bigBlind)
      ..maxPlayers = maxPlayers
      ..minPlayers = minPlayers
      ..currentPlayers = currentPlayers
      ..buyIn = Int64(buyIn)
      ..gameStarted = gameStarted
      ..allPlayersReady = allPlayersReady;
    // Include players if present (from notifications)
    if (players != null) {
      t.players.addAll(players!.map((p) => p.toProtobuf()));
    }
    return t;
  }
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
  @JsonKey(name: 'buy_in')
  final int buyIn;
  @JsonKey(name: 'starting_chips')
  final int startingChips;
  @JsonKey(name: 'time_bank_seconds')
  final int timeBankSeconds;
  @JsonKey(name: 'auto_start_ms')
  final int autoStartMs;
  @JsonKey(name: 'auto_advance_ms')
  final int autoAdvanceMs;

  CreatePokerTableArgs(
    this.smallBlind,
    this.bigBlind,
    this.maxPlayers,
    this.minPlayers,
    this.buyIn,
    this.startingChips,
    this.timeBankSeconds,
    this.autoStartMs,
    this.autoAdvanceMs,
  );

  factory CreatePokerTableArgs.fromJson(Map<String, dynamic> json) =>
      _$CreatePokerTableArgsFromJson(json);
  Map<String, dynamic> toJson() => _$CreatePokerTableArgsToJson(this);
}

@JsonSerializable()
class MakeBetArgs {
  @JsonKey(name: 'amount')
  final int amount;

  MakeBetArgs(this.amount);

  factory MakeBetArgs.fromJson(Map<String, dynamic> json) =>
      _$MakeBetArgsFromJson(json);
  Map<String, dynamic> toJson() => _$MakeBetArgsToJson(this);
}

@JsonSerializable()
class EvaluateHandArgs {
  @JsonKey(name: 'cards')
  final List<CardArg> cards;

  EvaluateHandArgs(this.cards);

  factory EvaluateHandArgs.fromJson(Map<String, dynamic> json) =>
      _$EvaluateHandArgsFromJson(json);
  Map<String, dynamic> toJson() => _$EvaluateHandArgsToJson(this);
}

@JsonSerializable()
class CardArg {
  @JsonKey(name: 'suit')
  final int suit;
  @JsonKey(name: 'value')
  final int value;

  CardArg(this.suit, this.value);

  factory CardArg.fromJson(Map<String, dynamic> json) =>
      _$CardArgFromJson(json);
  Map<String, dynamic> toJson() => _$CardArgToJson(this);
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
class CardDTO {
  @JsonKey(name: 'suit')
  final String suit;
  @JsonKey(name: 'value')
  final String value;

  CardDTO(this.suit, this.value);

  factory CardDTO.fromJson(Map<String, dynamic> json) =>
      _$CardDTOFromJson(json);
  Map<String, dynamic> toJson() => _$CardDTOToJson(this);

  pr.Card toProtobuf() {
    return pr.Card()
      ..suit = suit
      ..value = value;
  }
}

@JsonSerializable()
class PlayerDTO {
  @JsonKey(name: 'id')
  final String id;
  @JsonKey(name: 'name')
  final String name;
  @JsonKey(name: 'balance')
  final int balance;
  @JsonKey(name: 'hand')
  final List<CardDTO> hand;
  @JsonKey(name: 'currentBet')
  final int currentBet;
  @JsonKey(name: 'folded')
  final bool folded;
  @JsonKey(name: 'isTurn')
  final bool isTurn;
  @JsonKey(name: 'isAllIn')
  final bool isAllIn;
  @JsonKey(name: 'isDealer')
  final bool isDealer;
  @JsonKey(name: 'isReady')
  final bool isReady;
  @JsonKey(name: 'disconnected')
  final bool disconnected;
  @JsonKey(name: 'handDescription')
  final String handDescription;
  @JsonKey(name: 'playerState')
  final int playerState;
  @JsonKey(name: 'isSmallBlind')
  final bool isSmallBlind;
  @JsonKey(name: 'isBigBlind')
  final bool isBigBlind;
  @JsonKey(name: 'escrowId', defaultValue: '')
  final String escrowId;
  @JsonKey(name: 'escrowReady', defaultValue: false)
  final bool escrowReady;
  @JsonKey(name: 'tableSeat', defaultValue: 0)
  final int tableSeat;

  PlayerDTO(
    this.id,
    this.name,
    this.balance,
    this.hand,
    this.currentBet,
    this.folded,
    this.isTurn,
    this.isAllIn,
    this.isDealer,
    this.isReady,
    this.disconnected,
    this.handDescription,
    this.playerState,
    this.isSmallBlind,
    this.isBigBlind,
    this.escrowId,
    this.escrowReady,
    this.tableSeat,
  );

  factory PlayerDTO.fromJson(Map<String, dynamic> json) =>
      _$PlayerDTOFromJson(json);
  Map<String, dynamic> toJson() => _$PlayerDTOToJson(this);

  pr.Player toProtobuf() {
    return pr.Player()
      ..id = id
      ..name = name
      ..balance = Int64(balance)
      ..hand.addAll(hand.map((c) => c.toProtobuf()))
      ..currentBet = Int64(currentBet)
      ..folded = folded
      ..isTurn = isTurn
      ..isAllIn = isAllIn
      ..isDealer = isDealer
      ..isReady = isReady
      ..isDisconnected = disconnected
      ..handDescription = handDescription
      ..playerState = pr.PlayerState.valueOf(playerState) ??
          pr.PlayerState.PLAYER_STATE_UNINITIALIZED
      ..isSmallBlind = isSmallBlind
      ..isBigBlind = isBigBlind
      ..escrowId = escrowId
      ..escrowReady = escrowReady
      ..tableSeat = tableSeat;
  }
}

@JsonSerializable()
class GameUpdateDTO {
  @JsonKey(name: 'tableId')
  final String tableId;
  @JsonKey(name: 'phase')
  final int phase;
  @JsonKey(name: 'players')
  final List<PlayerDTO> players;
  @JsonKey(name: 'communityCards')
  final List<CardDTO> communityCards;
  @JsonKey(name: 'pot')
  final int pot;
  @JsonKey(name: 'currentBet')
  final int currentBet;
  @JsonKey(name: 'currentPlayer')
  final String currentPlayer;
  @JsonKey(name: 'minRaise')
  final int minRaise;
  @JsonKey(name: 'maxRaise')
  final int maxRaise;
  @JsonKey(name: 'gameStarted')
  final bool gameStarted;
  @JsonKey(name: 'playersRequired')
  final int playersRequired;
  @JsonKey(name: 'playersJoined')
  final int playersJoined;
  @JsonKey(name: 'phaseName')
  final String phaseName;
  @JsonKey(name: 'timeBankSeconds')
  final int timeBankSeconds;
  @JsonKey(name: 'turnDeadlineUnixMs')
  final int turnDeadlineUnixMs;
  @JsonKey(name: 'smallBlind', defaultValue: 0)
  final int smallBlind;
  @JsonKey(name: 'bigBlind', defaultValue: 0)
  final int bigBlind;

  GameUpdateDTO(
    this.tableId,
    this.phase,
    this.players,
    this.communityCards,
    this.pot,
    this.currentBet,
    this.currentPlayer,
    this.minRaise,
    this.maxRaise,
    this.gameStarted,
    this.playersRequired,
    this.playersJoined,
    this.phaseName,
    this.timeBankSeconds,
    this.turnDeadlineUnixMs,
    this.smallBlind,
    this.bigBlind,
  );

  factory GameUpdateDTO.fromJson(Map<String, dynamic> json) =>
      _$GameUpdateDTOFromJson(json);
  Map<String, dynamic> toJson() => _$GameUpdateDTOToJson(this);

  pr.GameUpdate toProtobuf() {
    return pr.GameUpdate()
      ..tableId = tableId
      ..phase = pr.GamePhase.valueOf(phase) ?? pr.GamePhase.WAITING
      ..players.addAll(players.map((p) => p.toProtobuf()))
      ..communityCards.addAll(communityCards.map((c) => c.toProtobuf()))
      ..pot = Int64(pot)
      ..currentBet = Int64(currentBet)
      ..currentPlayer = currentPlayer
      ..minRaise = Int64(minRaise)
      ..maxRaise = Int64(maxRaise)
      ..gameStarted = gameStarted
      ..playersRequired = playersRequired
      ..playersJoined = playersJoined
      ..phaseName = phaseName
      ..timeBankSeconds = timeBankSeconds
      ..turnDeadlineUnixMs = Int64(turnDeadlineUnixMs)
      ..smallBlind = Int64(smallBlind)
      ..bigBlind = Int64(bigBlind);
  }
}

@JsonSerializable()
class NotificationDTO {
  @JsonKey(name: 'type')
  final int type;
  @JsonKey(name: 'message')
  final String? message;
  @JsonKey(name: 'tableId')
  final String? tableId;
  @JsonKey(name: 'playerId')
  final String? playerId;
  @JsonKey(name: 'amount')
  final int? amount;
  @JsonKey(name: 'newBalance')
  final int? newBalance;
  @JsonKey(name: 'ready')
  final bool? ready;
  @JsonKey(name: 'started')
  final bool? started;
  @JsonKey(name: 'gameReadyToPlay')
  final bool? gameReadyToPlay;
  @JsonKey(name: 'countdown')
  final int? countdown;
  @JsonKey(name: 'table')
  final PokerTable? table;

  NotificationDTO(
    this.type, {
    this.message,
    this.tableId,
    this.playerId,
    this.amount,
    this.newBalance,
    this.ready,
    this.started,
    this.gameReadyToPlay,
    this.countdown,
    this.table,
  });

  factory NotificationDTO.fromJson(Map<String, dynamic> json) =>
      _$NotificationDTOFromJson(json);
  Map<String, dynamic> toJson() => _$NotificationDTOToJson(this);

  pr.Notification toProtobuf() {
    final n = pr.Notification()
      ..type = pr.NotificationType.valueOf(type) ?? pr.NotificationType.UNKNOWN;
    if (message != null) n.message = message!;
    if (tableId != null) n.tableId = tableId!;
    if (playerId != null) n.playerId = playerId!;
    if (amount != null) n.amount = Int64(amount!);
    if (newBalance != null) n.newBalance = Int64(newBalance!);
    if (ready != null) n.ready = ready!;
    if (started != null) n.started = started!;
    if (gameReadyToPlay != null) n.gameReadyToPlay = gameReadyToPlay!;
    if (countdown != null) n.countdown = countdown!;
    // Include table snapshot if present (for PLAYER_JOINED, PLAYER_LEFT, etc.)
    if (table != null) {
      n.table = table!.toProtobuf();
    }
    return n;
  }
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

  ZipLogsArgs(
    this.includeGolib,
    this.includeLn,
    this.onlyLastFile,
    this.destPath,
  );
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

/// PresignError represents an error that occurred during auto-presign
class PresignError {
  final String tableId;
  final String playerId;
  final String error;

  PresignError({
    required this.tableId,
    required this.playerId,
    required this.error,
  });

  factory PresignError.fromJson(Map<String, dynamic> json) {
    return PresignError(
      tableId: json['tableId'] as String? ?? '',
      playerId: json['playerId'] as String? ?? '',
      error: json['error'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'tableId': tableId,
      'playerId': playerId,
      'error': error,
    };
  }
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

  // Poker notifications from golib (poker.Notification proto)
  final StreamController<pr.Notification> ntfPokerNotifications =
      StreamController<pr.Notification>.broadcast();
  Stream<pr.Notification> pokerNotifications() => ntfPokerNotifications.stream;

  final StreamController<pr.GameUpdate> ntfGameUpdates =
      StreamController<pr.GameUpdate>.broadcast();
  Stream<pr.GameUpdate> gameUpdates() => ntfGameUpdates.stream;

  final StreamController<PresignError> ntfPresignErrors =
      StreamController<PresignError>.broadcast();
  Stream<PresignError> presignErrors() => ntfPresignErrors.stream;

  void disposeNtfStreams() {
    ntfAcceptedInvites.close();
    ntfLogLines.close();
    ntfRescanProgress.close();
    ntfUINotifications.close();
    ntfWaitingRoomCreated.close();
    ntfPokerNotifications.close();
    ntfGameUpdates.close();
    ntfPresignErrors.close();
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
            final wr = LocalWaitingRoom.fromJson(
              Map<String, dynamic>.from(data as Map),
            );
            ntfWaitingRoomCreated.add(wr);
          }
        } catch (e) {
          debugPrint("Failed to parse NTWRCreated payload: $e");
        }
        break;
      case NTPokerNotification:
        try {
          if (jsonPayload.isNotEmpty) {
            final data = jsonDecode(jsonPayload) as Map<String, dynamic>;
            final dto = NotificationDTO.fromJson(data);
            ntfPokerNotifications.add(dto.toProtobuf());
          }
        } catch (e, stackTrace) {
          debugPrint("Failed to parse NTPokerNotification payload: $e");
          debugPrint("Stack trace: $stackTrace");
          debugPrint("Payload was: $jsonPayload");
        }
        break;
      case NTGameUpdate:
        try {
          if (jsonPayload.isNotEmpty) {
            final data = jsonDecode(jsonPayload) as Map<String, dynamic>;
            final dto = GameUpdateDTO.fromJson(data);
            ntfGameUpdates.add(dto.toProtobuf());
          }
        } catch (e, stackTrace) {
          debugPrint("Failed to parse NTGameUpdate payload: $e");
          debugPrint("Stack trace: $stackTrace");
          debugPrint("Payload was: $jsonPayload");
        }
        break;
      case NTPresignError:
        try {
          if (jsonPayload.isNotEmpty) {
            final data = jsonDecode(jsonPayload) as Map<String, dynamic>;
            final error = PresignError.fromJson(data);
            ntfPresignErrors.add(error);
          }
        } catch (e, stackTrace) {
          debugPrint("Failed to parse NTPresignError payload: $e");
          debugPrint("Stack trace: $stackTrace");
          debugPrint("Payload was: $jsonPayload");
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
  Stream<pr.Notification> pokerNotifications();
  Stream<pr.GameUpdate> gameUpdates();

  Future<String> asyncHello(String name) async {
    final r = await asyncCall(CTHello, name);
    return r as String;
    // If platform returns non-string, this will throw; that’s desirable.
  }

  Future<void> register(RegisterRequest req) async {
    const cmdType = 0x24; // CTRegister
    await asyncCall(cmdType, req.toJson());
  }

  Future<LoginResponse> login(LoginRequest req) async {
    const cmdType = 0x25; // CTLogin
    final result = await asyncCall(cmdType, req.toJson());
    return LoginResponse.fromJson(result as Map<String, dynamic>);
  }

  Future<RequestLoginCodeResponse> requestLoginCode() async {
    const cmdType = CTRequestLoginCode;
    final result = await asyncCall(cmdType, {});
    return RequestLoginCodeResponse.fromJson(_asJsonMap(result));
  }

  Future<SetPayoutAddressResponse> setPayoutAddress(
      SetPayoutAddressRequest req) async {
    const cmdType = CTSetPayoutAddress;
    final result = await asyncCall(cmdType, req.toJson());
    return SetPayoutAddressResponse.fromJson(_asJsonMap(result));
  }

  Future<LoginResponse?> resumeSession() async {
    const cmdType = CTResumeSession;
    final result = await asyncCall(cmdType, {});
    if (result == null) return null;
    return LoginResponse.fromJson(_asJsonMap(result));
  }

  Future<void> logout() async {
    const cmdType = 0x26; // CTLogout
    await asyncCall(cmdType, {});
  }

  Future<LocalInfo> initClient(InitClient args) async {
    final res = await asyncCall(CTInitClient, args.toJson());
    return LocalInfo.fromJson(_asJsonMap(res));
  }

  Future<Map<String, dynamic>> createDefaultConfig(
    CreateDefaultConfig args,
  ) async {
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

  Future<LocalWaitingRoom> JoinWaitingRoom(
    String id, {
    String? escrowId,
  }) async {
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

  Future<Map<String, String>> deriveSettlementSessionKey(int index) async {
    final res = await asyncCall(CTDeriveSessionKey, {'index': index});
    final m = _asJsonMap(res);
    return m.map((k, v) => MapEntry(k, v == null ? '' : v.toString()));
  }

  Future<Map<String, dynamic>> openEscrow({
    required int betAtoms,
    required String compPubkey,
    required int keyIndex,
    int csvBlocks = 64,
  }) async {
    final payload = {
      'bet_atoms': betAtoms,
      'csv_blocks': csvBlocks,
      'comp_pubkey': compPubkey,
      'key_index': keyIndex,
    };
    final res = await asyncCall(CTOpenEscrow, payload);
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> getEscrowStatus(String escrowId) async {
    final res = await asyncCall(CTGetEscrowStatus, {'escrow_id': escrowId});
    return _asJsonMap(res);
  }

  Future<List<dynamic>> getEscrowHistory() async {
    final res = await asyncCall(CTGetEscrowHistory, {});
    return _asJsonListOrWrap(res);
  }

  Future<List<dynamic>> getBindableEscrows() async {
    final res = await asyncCall(CTGetBindableEscrows, {});
    return _asJsonListOrWrap(res);
  }

  Future<String> getPayoutAddress() async {
    final res = await asyncCall(CTGetPayoutAddress, {});
    final m = _asJsonMap(res);
    final v = m['payout_address'];
    return v == null ? '' : v.toString();
  }

  Future<Map<String, dynamic>> refundEscrow({
    required String escrowId,
    required String destAddr,
    int feeAtoms = 20000,
    int csvBlocks = 64,
    int? utxoValue,
  }) async {
    final payload = {
      'escrow_id': escrowId,
      'dest_addr': destAddr,
      'fee_atoms': feeAtoms,
      'csv_blocks': csvBlocks,
      if (utxoValue != null && utxoValue > 0) 'utxo_value': utxoValue,
    };
    final res = await asyncCall(CTRefundEscrow, payload);
    return _asJsonMap(res);
  }

  Future<void> updateEscrowHistory(Map<String, dynamic> escrowInfo) async {
    await asyncCall(CTUpdateEscrowHistory, escrowInfo);
  }

  Future<void> deleteEscrowHistory(String escrowId) async {
    await asyncCall(CTDeleteEscrowHistory, {'escrow_id': escrowId});
  }

  Future<Map<String, dynamic>> getEscrowById(String escrowId) async {
    final res = await asyncCall(CTGetEscrowById, {'escrow_id': escrowId});
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> bindEscrow({
    required String tableId,
    required int seatIndex,
    required String outpoint,
    String matchId = '',
  }) async {
    final res = await asyncCall(CTBindEscrow, {
      'table_id': tableId,
      'seat_index': seatIndex,
      'outpoint': outpoint,
      'match_id': matchId,
    });
    return _asJsonMap(res);
  }

  Future<void> startPreSign({
    required String matchId,
    required String tableId,
    required String escrowId,
    required String compPriv,
  }) async {
    await asyncCall(CTStartPreSign, {
      'match_id': matchId,
      'table_id': tableId,
      'escrow_id': escrowId,
      'comp_priv': compPriv,
    });
  }

  Future<Map<String, String>> archiveSessionKey(String matchId, Map<String, dynamic> escrowInfo) async {
    final res = await asyncCall(CTArchiveSessionKey, {
      'match_id': matchId,
      'escrow_info': escrowInfo,
    });
    return _asJsonMap(res).map((k, v) => MapEntry(k, v.toString()));
  }

  Future<void> archiveSettlementSessionKey(String matchId) async {
    await asyncCall(CTArchiveSessionKey, {'match_id': matchId});
  }

  // Poker table methods
  Future<List<PokerTable>> getPokerTables() async {
    final res = await asyncCall(CTGetPokerTables, "");
    if (res == null) return [];

    final list = _asJsonListOrWrap(res);

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

  Future<Map<String, dynamic>> createPokerTable(
    CreatePokerTableArgs args,
  ) async {
    final res = await asyncCall(CTCreatePokerTable, args.toJson());
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> leavePokerTable() async {
    final res = await asyncCall(CTLeavePokerTable, "");
    return _asJsonMap(res);
  }

  Future<String> getPokerCurrentTable() async {
    final res = await asyncCall(CTGetPlayerCurrentTable, "");
    final m = _asJsonMap(res);
    return m['table_id'] as String? ?? "";
  }

  Future<void> showCards() async {
    await asyncCall(CTShowCards, "");
  }

  Future<void> hideCards() async {
    await asyncCall(CTHideCards, "");
  }

  Future<void> makeBet(MakeBetArgs args) async {
    await asyncCall(CTMakeBet, args.toJson());
  }

  Future<void> callBet() async {
    await asyncCall(CTCallBet, "");
  }

  Future<void> foldBet() async {
    await asyncCall(CTFoldBet, "");
  }

  Future<void> checkBet() async {
    await asyncCall(CTCheckBet, "");
  }

  Future<void> setPlayerReady() async {
    await asyncCall(CTSetPlayerReady, "");
  }

  Future<void> setPlayerUnready() async {
    await asyncCall(CTSetPlayerUnready, "");
  }

  Future<Map<String, dynamic>> getGameState() async {
    final res = await asyncCall(CTGetGameState, "");
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> getLastWinners() async {
    final res = await asyncCall(CTGetLastWinners, "");
    return _asJsonMap(res);
  }

  Future<Map<String, dynamic>> evaluateHand(EvaluateHandArgs args) async {
    final res = await asyncCall(CTEvaluateHand, args.toJson());
    return _asJsonMap(res);
  }

  Future<void> startGameStream() async {
    await asyncCall(CTStartGameStream, "");
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
const int CTOpenEscrow = 0x0b;
const int CTStartPreSign = 0x0c;
const int CTBindEscrow = 0x0d;
const int CTArchiveSessionKey = 0x0e;
const int CTDeriveSessionKey = 0x0f;
const int CTGetEscrowStatus = 0x30;
const int CTGetEscrowHistory = 0x31;
const int CTGetFinalizeBundle = 0x32;
const int CTGetEscrowById = 0x33;
const int CTGetBindableEscrows = 0x34;
const int CTRefundEscrow = 0x35;
const int CTUpdateEscrowHistory = 0x36;
const int CTDeleteEscrowHistory = 0x37;

// Poker-specific commands
const int CTGetPlayerCurrentTable = 0x10;
const int CTLoadConfig = 0x11;
const int CTGetPokerTables = 0x12;
const int CTJoinPokerTable = 0x13;
const int CTCreatePokerTable = 0x14;
const int CTLeavePokerTable = 0x15;
const int CTCreateDefaultConfig = 0x17;
const int CTCreateDefaultServerCert = 0x18;
const int CTShowCards = 0x19;
const int CTHideCards = 0x1a;
const int CTMakeBet = 0x1b;
const int CTCallBet = 0x1c;
const int CTFoldBet = 0x1d;
const int CTCheckBet = 0x1e;
const int CTGetGameState = 0x1f;
const int CTGetLastWinners = 0x20;
const int CTEvaluateHand = 0x21;
const int CTSetPlayerReady = 0x22;
const int CTSetPlayerUnready = 0x23;
const int CTStartGameStream = 0x27;

const int CTCloseLockFile = 0x60;
// Auth commands
const int CTRegister = 0x24;
const int CTLogin = 0x25;
const int CTLogout = 0x26;
const int CTResumeSession = 0x28;
const int CTRequestLoginCode = 0x29;
const int CTSetPayoutAddress = 0x2a;
const int CTGetPayoutAddress = 0x2b;

const int notificationsStartID = 0x1000;
const int notificationClientStopped =
    0x1001; // kept for compatibility; actual client-stopped is 0x1002 in golib
const int NTNOP = 0x1004;
const int NTWRCreated = 0x1005;
const int NTPokerNotification = 0x1006;
const int NTGameUpdate = 0x1007;
const int NTPresignError = 0x1008;
