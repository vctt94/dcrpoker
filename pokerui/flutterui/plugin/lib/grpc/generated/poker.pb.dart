// This is a generated file - do not edit.
//
// Generated from poker.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'poker.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'poker.pbenum.dart';

/// Game Messages
class StartGameStreamRequest extends $pb.GeneratedMessage {
  factory StartGameStreamRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  StartGameStreamRequest._();

  factory StartGameStreamRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartGameStreamRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartGameStreamRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartGameStreamRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartGameStreamRequest copyWith(
          void Function(StartGameStreamRequest) updates) =>
      super.copyWith((message) => updates(message as StartGameStreamRequest))
          as StartGameStreamRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartGameStreamRequest create() => StartGameStreamRequest._();
  @$core.override
  StartGameStreamRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartGameStreamRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartGameStreamRequest>(create);
  static StartGameStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class GameUpdate extends $pb.GeneratedMessage {
  factory GameUpdate({
    $core.String? tableId,
    GamePhase? phase,
    $core.Iterable<Player>? players,
    $core.Iterable<Card>? communityCards,
    $fixnum.Int64? pot,
    $fixnum.Int64? currentBet,
    $core.String? currentPlayer,
    $fixnum.Int64? minRaise,
    $fixnum.Int64? maxRaise,
    $core.bool? gameStarted,
    $core.int? playersRequired,
    $core.int? playersJoined,
    $core.String? phaseName,
    $core.int? timeBankSeconds,
    $fixnum.Int64? turnDeadlineUnixMs,
    $fixnum.Int64? smallBlind,
    $fixnum.Int64? bigBlind,
  }) {
    final result = create();
    if (tableId != null) result.tableId = tableId;
    if (phase != null) result.phase = phase;
    if (players != null) result.players.addAll(players);
    if (communityCards != null) result.communityCards.addAll(communityCards);
    if (pot != null) result.pot = pot;
    if (currentBet != null) result.currentBet = currentBet;
    if (currentPlayer != null) result.currentPlayer = currentPlayer;
    if (minRaise != null) result.minRaise = minRaise;
    if (maxRaise != null) result.maxRaise = maxRaise;
    if (gameStarted != null) result.gameStarted = gameStarted;
    if (playersRequired != null) result.playersRequired = playersRequired;
    if (playersJoined != null) result.playersJoined = playersJoined;
    if (phaseName != null) result.phaseName = phaseName;
    if (timeBankSeconds != null) result.timeBankSeconds = timeBankSeconds;
    if (turnDeadlineUnixMs != null)
      result.turnDeadlineUnixMs = turnDeadlineUnixMs;
    if (smallBlind != null) result.smallBlind = smallBlind;
    if (bigBlind != null) result.bigBlind = bigBlind;
    return result;
  }

  GameUpdate._();

  factory GameUpdate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GameUpdate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GameUpdate',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..aE<GamePhase>(2, _omitFieldNames ? '' : 'phase',
        enumValues: GamePhase.values)
    ..pPM<Player>(3, _omitFieldNames ? '' : 'players',
        subBuilder: Player.create)
    ..pPM<Card>(4, _omitFieldNames ? '' : 'communityCards',
        subBuilder: Card.create)
    ..aInt64(5, _omitFieldNames ? '' : 'pot')
    ..aInt64(6, _omitFieldNames ? '' : 'currentBet')
    ..aOS(7, _omitFieldNames ? '' : 'currentPlayer')
    ..aInt64(8, _omitFieldNames ? '' : 'minRaise')
    ..aInt64(9, _omitFieldNames ? '' : 'maxRaise')
    ..aOB(10, _omitFieldNames ? '' : 'gameStarted')
    ..aI(11, _omitFieldNames ? '' : 'playersRequired')
    ..aI(12, _omitFieldNames ? '' : 'playersJoined')
    ..aOS(13, _omitFieldNames ? '' : 'phaseName')
    ..aI(14, _omitFieldNames ? '' : 'timeBankSeconds')
    ..aInt64(15, _omitFieldNames ? '' : 'turnDeadlineUnixMs')
    ..aInt64(16, _omitFieldNames ? '' : 'smallBlind')
    ..aInt64(17, _omitFieldNames ? '' : 'bigBlind')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GameUpdate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GameUpdate copyWith(void Function(GameUpdate) updates) =>
      super.copyWith((message) => updates(message as GameUpdate)) as GameUpdate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GameUpdate create() => GameUpdate._();
  @$core.override
  GameUpdate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GameUpdate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GameUpdate>(create);
  static GameUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => $_clearField(1);

  @$pb.TagNumber(2)
  GamePhase get phase => $_getN(1);
  @$pb.TagNumber(2)
  set phase(GamePhase value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPhase() => $_has(1);
  @$pb.TagNumber(2)
  void clearPhase() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<Player> get players => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbList<Card> get communityCards => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get pot => $_getI64(4);
  @$pb.TagNumber(5)
  set pot($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPot() => $_has(4);
  @$pb.TagNumber(5)
  void clearPot() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get currentBet => $_getI64(5);
  @$pb.TagNumber(6)
  set currentBet($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCurrentBet() => $_has(5);
  @$pb.TagNumber(6)
  void clearCurrentBet() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get currentPlayer => $_getSZ(6);
  @$pb.TagNumber(7)
  set currentPlayer($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCurrentPlayer() => $_has(6);
  @$pb.TagNumber(7)
  void clearCurrentPlayer() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get minRaise => $_getI64(7);
  @$pb.TagNumber(8)
  set minRaise($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasMinRaise() => $_has(7);
  @$pb.TagNumber(8)
  void clearMinRaise() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get maxRaise => $_getI64(8);
  @$pb.TagNumber(9)
  set maxRaise($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasMaxRaise() => $_has(8);
  @$pb.TagNumber(9)
  void clearMaxRaise() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get gameStarted => $_getBF(9);
  @$pb.TagNumber(10)
  set gameStarted($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasGameStarted() => $_has(9);
  @$pb.TagNumber(10)
  void clearGameStarted() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get playersRequired => $_getIZ(10);
  @$pb.TagNumber(11)
  set playersRequired($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPlayersRequired() => $_has(10);
  @$pb.TagNumber(11)
  void clearPlayersRequired() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.int get playersJoined => $_getIZ(11);
  @$pb.TagNumber(12)
  set playersJoined($core.int value) => $_setSignedInt32(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPlayersJoined() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlayersJoined() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get phaseName => $_getSZ(12);
  @$pb.TagNumber(13)
  set phaseName($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasPhaseName() => $_has(12);
  @$pb.TagNumber(13)
  void clearPhaseName() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.int get timeBankSeconds => $_getIZ(13);
  @$pb.TagNumber(14)
  set timeBankSeconds($core.int value) => $_setSignedInt32(13, value);
  @$pb.TagNumber(14)
  $core.bool hasTimeBankSeconds() => $_has(13);
  @$pb.TagNumber(14)
  void clearTimeBankSeconds() => $_clearField(14);

  @$pb.TagNumber(15)
  $fixnum.Int64 get turnDeadlineUnixMs => $_getI64(14);
  @$pb.TagNumber(15)
  set turnDeadlineUnixMs($fixnum.Int64 value) => $_setInt64(14, value);
  @$pb.TagNumber(15)
  $core.bool hasTurnDeadlineUnixMs() => $_has(14);
  @$pb.TagNumber(15)
  void clearTurnDeadlineUnixMs() => $_clearField(15);

  @$pb.TagNumber(16)
  $fixnum.Int64 get smallBlind => $_getI64(15);
  @$pb.TagNumber(16)
  set smallBlind($fixnum.Int64 value) => $_setInt64(15, value);
  @$pb.TagNumber(16)
  $core.bool hasSmallBlind() => $_has(15);
  @$pb.TagNumber(16)
  void clearSmallBlind() => $_clearField(16);

  @$pb.TagNumber(17)
  $fixnum.Int64 get bigBlind => $_getI64(16);
  @$pb.TagNumber(17)
  set bigBlind($fixnum.Int64 value) => $_setInt64(16, value);
  @$pb.TagNumber(17)
  $core.bool hasBigBlind() => $_has(16);
  @$pb.TagNumber(17)
  void clearBigBlind() => $_clearField(17);
}

class MakeBetRequest extends $pb.GeneratedMessage {
  factory MakeBetRequest({
    $core.String? playerId,
    $core.String? tableId,
    $fixnum.Int64? amount,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    if (amount != null) result.amount = amount;
    return result;
  }

  MakeBetRequest._();

  factory MakeBetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory MakeBetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MakeBetRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..aInt64(3, _omitFieldNames ? '' : 'amount')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MakeBetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MakeBetRequest copyWith(void Function(MakeBetRequest) updates) =>
      super.copyWith((message) => updates(message as MakeBetRequest))
          as MakeBetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MakeBetRequest create() => MakeBetRequest._();
  @$core.override
  MakeBetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static MakeBetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MakeBetRequest>(create);
  static MakeBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get amount => $_getI64(2);
  @$pb.TagNumber(3)
  set amount($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAmount() => $_has(2);
  @$pb.TagNumber(3)
  void clearAmount() => $_clearField(3);
}

class MakeBetResponse extends $pb.GeneratedMessage {
  factory MakeBetResponse({
    $core.bool? success,
    $core.String? message,
    $fixnum.Int64? newBalance,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    if (newBalance != null) result.newBalance = newBalance;
    return result;
  }

  MakeBetResponse._();

  factory MakeBetResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory MakeBetResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MakeBetResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aInt64(3, _omitFieldNames ? '' : 'newBalance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MakeBetResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MakeBetResponse copyWith(void Function(MakeBetResponse) updates) =>
      super.copyWith((message) => updates(message as MakeBetResponse))
          as MakeBetResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MakeBetResponse create() => MakeBetResponse._();
  @$core.override
  MakeBetResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static MakeBetResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MakeBetResponse>(create);
  static MakeBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get newBalance => $_getI64(2);
  @$pb.TagNumber(3)
  set newBalance($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNewBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewBalance() => $_clearField(3);
}

class FoldBetRequest extends $pb.GeneratedMessage {
  factory FoldBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  FoldBetRequest._();

  factory FoldBetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FoldBetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FoldBetRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FoldBetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FoldBetRequest copyWith(void Function(FoldBetRequest) updates) =>
      super.copyWith((message) => updates(message as FoldBetRequest))
          as FoldBetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FoldBetRequest create() => FoldBetRequest._();
  @$core.override
  FoldBetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FoldBetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FoldBetRequest>(create);
  static FoldBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class FoldBetResponse extends $pb.GeneratedMessage {
  factory FoldBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  FoldBetResponse._();

  factory FoldBetResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FoldBetResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FoldBetResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FoldBetResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FoldBetResponse copyWith(void Function(FoldBetResponse) updates) =>
      super.copyWith((message) => updates(message as FoldBetResponse))
          as FoldBetResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FoldBetResponse create() => FoldBetResponse._();
  @$core.override
  FoldBetResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FoldBetResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FoldBetResponse>(create);
  static FoldBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class CheckBetRequest extends $pb.GeneratedMessage {
  factory CheckBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  CheckBetRequest._();

  factory CheckBetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CheckBetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CheckBetRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CheckBetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CheckBetRequest copyWith(void Function(CheckBetRequest) updates) =>
      super.copyWith((message) => updates(message as CheckBetRequest))
          as CheckBetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CheckBetRequest create() => CheckBetRequest._();
  @$core.override
  CheckBetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CheckBetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CheckBetRequest>(create);
  static CheckBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class CheckBetResponse extends $pb.GeneratedMessage {
  factory CheckBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  CheckBetResponse._();

  factory CheckBetResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CheckBetResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CheckBetResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CheckBetResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CheckBetResponse copyWith(void Function(CheckBetResponse) updates) =>
      super.copyWith((message) => updates(message as CheckBetResponse))
          as CheckBetResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CheckBetResponse create() => CheckBetResponse._();
  @$core.override
  CheckBetResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CheckBetResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CheckBetResponse>(create);
  static CheckBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class CallBetRequest extends $pb.GeneratedMessage {
  factory CallBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  CallBetRequest._();

  factory CallBetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CallBetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CallBetRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallBetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallBetRequest copyWith(void Function(CallBetRequest) updates) =>
      super.copyWith((message) => updates(message as CallBetRequest))
          as CallBetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CallBetRequest create() => CallBetRequest._();
  @$core.override
  CallBetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CallBetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CallBetRequest>(create);
  static CallBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class CallBetResponse extends $pb.GeneratedMessage {
  factory CallBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  CallBetResponse._();

  factory CallBetResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CallBetResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CallBetResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallBetResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallBetResponse copyWith(void Function(CallBetResponse) updates) =>
      super.copyWith((message) => updates(message as CallBetResponse))
          as CallBetResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CallBetResponse create() => CallBetResponse._();
  @$core.override
  CallBetResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CallBetResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CallBetResponse>(create);
  static CallBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class GetGameStateRequest extends $pb.GeneratedMessage {
  factory GetGameStateRequest({
    $core.String? tableId,
    $core.String? playerId,
  }) {
    final result = create();
    if (tableId != null) result.tableId = tableId;
    if (playerId != null) result.playerId = playerId;
    return result;
  }

  GetGameStateRequest._();

  factory GetGameStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetGameStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetGameStateRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..aOS(2, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetGameStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetGameStateRequest copyWith(void Function(GetGameStateRequest) updates) =>
      super.copyWith((message) => updates(message as GetGameStateRequest))
          as GetGameStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetGameStateRequest create() => GetGameStateRequest._();
  @$core.override
  GetGameStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetGameStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetGameStateRequest>(create);
  static GetGameStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get playerId => $_getSZ(1);
  @$pb.TagNumber(2)
  set playerId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPlayerId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPlayerId() => $_clearField(2);
}

class GetGameStateResponse extends $pb.GeneratedMessage {
  factory GetGameStateResponse({
    GameUpdate? gameState,
  }) {
    final result = create();
    if (gameState != null) result.gameState = gameState;
    return result;
  }

  GetGameStateResponse._();

  factory GetGameStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetGameStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetGameStateResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOM<GameUpdate>(1, _omitFieldNames ? '' : 'gameState',
        subBuilder: GameUpdate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetGameStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetGameStateResponse copyWith(void Function(GetGameStateResponse) updates) =>
      super.copyWith((message) => updates(message as GetGameStateResponse))
          as GetGameStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetGameStateResponse create() => GetGameStateResponse._();
  @$core.override
  GetGameStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetGameStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetGameStateResponse>(create);
  static GetGameStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  GameUpdate get gameState => $_getN(0);
  @$pb.TagNumber(1)
  set gameState(GameUpdate value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasGameState() => $_has(0);
  @$pb.TagNumber(1)
  void clearGameState() => $_clearField(1);
  @$pb.TagNumber(1)
  GameUpdate ensureGameState() => $_ensure(0);
}

class EvaluateHandRequest extends $pb.GeneratedMessage {
  factory EvaluateHandRequest({
    $core.Iterable<Card>? cards,
  }) {
    final result = create();
    if (cards != null) result.cards.addAll(cards);
    return result;
  }

  EvaluateHandRequest._();

  factory EvaluateHandRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EvaluateHandRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EvaluateHandRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..pPM<Card>(1, _omitFieldNames ? '' : 'cards', subBuilder: Card.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EvaluateHandRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EvaluateHandRequest copyWith(void Function(EvaluateHandRequest) updates) =>
      super.copyWith((message) => updates(message as EvaluateHandRequest))
          as EvaluateHandRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EvaluateHandRequest create() => EvaluateHandRequest._();
  @$core.override
  EvaluateHandRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EvaluateHandRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EvaluateHandRequest>(create);
  static EvaluateHandRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Card> get cards => $_getList(0);
}

class EvaluateHandResponse extends $pb.GeneratedMessage {
  factory EvaluateHandResponse({
    HandRank? rank,
    $core.String? description,
    $core.Iterable<Card>? bestHand,
  }) {
    final result = create();
    if (rank != null) result.rank = rank;
    if (description != null) result.description = description;
    if (bestHand != null) result.bestHand.addAll(bestHand);
    return result;
  }

  EvaluateHandResponse._();

  factory EvaluateHandResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EvaluateHandResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EvaluateHandResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aE<HandRank>(1, _omitFieldNames ? '' : 'rank',
        enumValues: HandRank.values)
    ..aOS(2, _omitFieldNames ? '' : 'description')
    ..pPM<Card>(3, _omitFieldNames ? '' : 'bestHand', subBuilder: Card.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EvaluateHandResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EvaluateHandResponse copyWith(void Function(EvaluateHandResponse) updates) =>
      super.copyWith((message) => updates(message as EvaluateHandResponse))
          as EvaluateHandResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EvaluateHandResponse create() => EvaluateHandResponse._();
  @$core.override
  EvaluateHandResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EvaluateHandResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EvaluateHandResponse>(create);
  static EvaluateHandResponse? _defaultInstance;

  @$pb.TagNumber(1)
  HandRank get rank => $_getN(0);
  @$pb.TagNumber(1)
  set rank(HandRank value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRank() => $_has(0);
  @$pb.TagNumber(1)
  void clearRank() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get description => $_getSZ(1);
  @$pb.TagNumber(2)
  set description($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDescription() => $_has(1);
  @$pb.TagNumber(2)
  void clearDescription() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<Card> get bestHand => $_getList(2);
}

class GetLastWinnersRequest extends $pb.GeneratedMessage {
  factory GetLastWinnersRequest({
    $core.String? tableId,
  }) {
    final result = create();
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  GetLastWinnersRequest._();

  factory GetLastWinnersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLastWinnersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLastWinnersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLastWinnersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLastWinnersRequest copyWith(
          void Function(GetLastWinnersRequest) updates) =>
      super.copyWith((message) => updates(message as GetLastWinnersRequest))
          as GetLastWinnersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLastWinnersRequest create() => GetLastWinnersRequest._();
  @$core.override
  GetLastWinnersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLastWinnersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLastWinnersRequest>(create);
  static GetLastWinnersRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => $_clearField(1);
}

class GetLastWinnersResponse extends $pb.GeneratedMessage {
  factory GetLastWinnersResponse({
    $core.Iterable<Winner>? winners,
  }) {
    final result = create();
    if (winners != null) result.winners.addAll(winners);
    return result;
  }

  GetLastWinnersResponse._();

  factory GetLastWinnersResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLastWinnersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLastWinnersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..pPM<Winner>(1, _omitFieldNames ? '' : 'winners',
        subBuilder: Winner.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLastWinnersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLastWinnersResponse copyWith(
          void Function(GetLastWinnersResponse) updates) =>
      super.copyWith((message) => updates(message as GetLastWinnersResponse))
          as GetLastWinnersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLastWinnersResponse create() => GetLastWinnersResponse._();
  @$core.override
  GetLastWinnersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLastWinnersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLastWinnersResponse>(create);
  static GetLastWinnersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Winner> get winners => $_getList(0);
}

class Winner extends $pb.GeneratedMessage {
  factory Winner({
    $core.String? playerId,
    HandRank? handRank,
    $core.Iterable<Card>? bestHand,
    $fixnum.Int64? winnings,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (handRank != null) result.handRank = handRank;
    if (bestHand != null) result.bestHand.addAll(bestHand);
    if (winnings != null) result.winnings = winnings;
    return result;
  }

  Winner._();

  factory Winner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Winner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Winner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aE<HandRank>(2, _omitFieldNames ? '' : 'handRank',
        enumValues: HandRank.values)
    ..pPM<Card>(3, _omitFieldNames ? '' : 'bestHand', subBuilder: Card.create)
    ..aInt64(4, _omitFieldNames ? '' : 'winnings')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Winner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Winner copyWith(void Function(Winner) updates) =>
      super.copyWith((message) => updates(message as Winner)) as Winner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Winner create() => Winner._();
  @$core.override
  Winner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Winner getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Winner>(create);
  static Winner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  HandRank get handRank => $_getN(1);
  @$pb.TagNumber(2)
  set handRank(HandRank value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasHandRank() => $_has(1);
  @$pb.TagNumber(2)
  void clearHandRank() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<Card> get bestHand => $_getList(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get winnings => $_getI64(3);
  @$pb.TagNumber(4)
  set winnings($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasWinnings() => $_has(3);
  @$pb.TagNumber(4)
  void clearWinnings() => $_clearField(4);
}

/// Lobby Messages
class CreateTableRequest extends $pb.GeneratedMessage {
  factory CreateTableRequest({
    $core.String? playerId,
    $fixnum.Int64? smallBlind,
    $fixnum.Int64? bigBlind,
    $core.int? maxPlayers,
    $core.int? minPlayers,
    $fixnum.Int64? minBalance,
    $fixnum.Int64? buyIn,
    $fixnum.Int64? startingChips,
    $core.int? timeBankSeconds,
    $core.int? autoStartMs,
    $core.int? autoAdvanceMs,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (smallBlind != null) result.smallBlind = smallBlind;
    if (bigBlind != null) result.bigBlind = bigBlind;
    if (maxPlayers != null) result.maxPlayers = maxPlayers;
    if (minPlayers != null) result.minPlayers = minPlayers;
    if (minBalance != null) result.minBalance = minBalance;
    if (buyIn != null) result.buyIn = buyIn;
    if (startingChips != null) result.startingChips = startingChips;
    if (timeBankSeconds != null) result.timeBankSeconds = timeBankSeconds;
    if (autoStartMs != null) result.autoStartMs = autoStartMs;
    if (autoAdvanceMs != null) result.autoAdvanceMs = autoAdvanceMs;
    return result;
  }

  CreateTableRequest._();

  factory CreateTableRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateTableRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateTableRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aInt64(2, _omitFieldNames ? '' : 'smallBlind')
    ..aInt64(3, _omitFieldNames ? '' : 'bigBlind')
    ..aI(4, _omitFieldNames ? '' : 'maxPlayers')
    ..aI(5, _omitFieldNames ? '' : 'minPlayers')
    ..aInt64(6, _omitFieldNames ? '' : 'minBalance')
    ..aInt64(7, _omitFieldNames ? '' : 'buyIn')
    ..aInt64(8, _omitFieldNames ? '' : 'startingChips')
    ..aI(9, _omitFieldNames ? '' : 'timeBankSeconds')
    ..aI(10, _omitFieldNames ? '' : 'autoStartMs')
    ..aI(11, _omitFieldNames ? '' : 'autoAdvanceMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateTableRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateTableRequest copyWith(void Function(CreateTableRequest) updates) =>
      super.copyWith((message) => updates(message as CreateTableRequest))
          as CreateTableRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateTableRequest create() => CreateTableRequest._();
  @$core.override
  CreateTableRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateTableRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateTableRequest>(create);
  static CreateTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get smallBlind => $_getI64(1);
  @$pb.TagNumber(2)
  set smallBlind($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSmallBlind() => $_has(1);
  @$pb.TagNumber(2)
  void clearSmallBlind() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get bigBlind => $_getI64(2);
  @$pb.TagNumber(3)
  set bigBlind($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBigBlind() => $_has(2);
  @$pb.TagNumber(3)
  void clearBigBlind() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get maxPlayers => $_getIZ(3);
  @$pb.TagNumber(4)
  set maxPlayers($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMaxPlayers() => $_has(3);
  @$pb.TagNumber(4)
  void clearMaxPlayers() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get minPlayers => $_getIZ(4);
  @$pb.TagNumber(5)
  set minPlayers($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMinPlayers() => $_has(4);
  @$pb.TagNumber(5)
  void clearMinPlayers() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get minBalance => $_getI64(5);
  @$pb.TagNumber(6)
  set minBalance($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMinBalance() => $_has(5);
  @$pb.TagNumber(6)
  void clearMinBalance() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get buyIn => $_getI64(6);
  @$pb.TagNumber(7)
  set buyIn($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasBuyIn() => $_has(6);
  @$pb.TagNumber(7)
  void clearBuyIn() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get startingChips => $_getI64(7);
  @$pb.TagNumber(8)
  set startingChips($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasStartingChips() => $_has(7);
  @$pb.TagNumber(8)
  void clearStartingChips() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.int get timeBankSeconds => $_getIZ(8);
  @$pb.TagNumber(9)
  set timeBankSeconds($core.int value) => $_setSignedInt32(8, value);
  @$pb.TagNumber(9)
  $core.bool hasTimeBankSeconds() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimeBankSeconds() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get autoStartMs => $_getIZ(9);
  @$pb.TagNumber(10)
  set autoStartMs($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasAutoStartMs() => $_has(9);
  @$pb.TagNumber(10)
  void clearAutoStartMs() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get autoAdvanceMs => $_getIZ(10);
  @$pb.TagNumber(11)
  set autoAdvanceMs($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasAutoAdvanceMs() => $_has(10);
  @$pb.TagNumber(11)
  void clearAutoAdvanceMs() => $_clearField(11);
}

class CreateTableResponse extends $pb.GeneratedMessage {
  factory CreateTableResponse({
    $core.String? tableId,
    $core.String? message,
  }) {
    final result = create();
    if (tableId != null) result.tableId = tableId;
    if (message != null) result.message = message;
    return result;
  }

  CreateTableResponse._();

  factory CreateTableResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateTableResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateTableResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateTableResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateTableResponse copyWith(void Function(CreateTableResponse) updates) =>
      super.copyWith((message) => updates(message as CreateTableResponse))
          as CreateTableResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateTableResponse create() => CreateTableResponse._();
  @$core.override
  CreateTableResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateTableResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateTableResponse>(create);
  static CreateTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class JoinTableRequest extends $pb.GeneratedMessage {
  factory JoinTableRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  JoinTableRequest._();

  factory JoinTableRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory JoinTableRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'JoinTableRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinTableRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinTableRequest copyWith(void Function(JoinTableRequest) updates) =>
      super.copyWith((message) => updates(message as JoinTableRequest))
          as JoinTableRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinTableRequest create() => JoinTableRequest._();
  @$core.override
  JoinTableRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static JoinTableRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<JoinTableRequest>(create);
  static JoinTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class JoinTableResponse extends $pb.GeneratedMessage {
  factory JoinTableResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  JoinTableResponse._();

  factory JoinTableResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory JoinTableResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'JoinTableResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinTableResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinTableResponse copyWith(void Function(JoinTableResponse) updates) =>
      super.copyWith((message) => updates(message as JoinTableResponse))
          as JoinTableResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinTableResponse create() => JoinTableResponse._();
  @$core.override
  JoinTableResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static JoinTableResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<JoinTableResponse>(create);
  static JoinTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class LeaveTableRequest extends $pb.GeneratedMessage {
  factory LeaveTableRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  LeaveTableRequest._();

  factory LeaveTableRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LeaveTableRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LeaveTableRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LeaveTableRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LeaveTableRequest copyWith(void Function(LeaveTableRequest) updates) =>
      super.copyWith((message) => updates(message as LeaveTableRequest))
          as LeaveTableRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LeaveTableRequest create() => LeaveTableRequest._();
  @$core.override
  LeaveTableRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LeaveTableRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LeaveTableRequest>(create);
  static LeaveTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class LeaveTableResponse extends $pb.GeneratedMessage {
  factory LeaveTableResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  LeaveTableResponse._();

  factory LeaveTableResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LeaveTableResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LeaveTableResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LeaveTableResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LeaveTableResponse copyWith(void Function(LeaveTableResponse) updates) =>
      super.copyWith((message) => updates(message as LeaveTableResponse))
          as LeaveTableResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LeaveTableResponse create() => LeaveTableResponse._();
  @$core.override
  LeaveTableResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LeaveTableResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LeaveTableResponse>(create);
  static LeaveTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class GetTablesRequest extends $pb.GeneratedMessage {
  factory GetTablesRequest() => create();

  GetTablesRequest._();

  factory GetTablesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTablesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTablesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTablesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTablesRequest copyWith(void Function(GetTablesRequest) updates) =>
      super.copyWith((message) => updates(message as GetTablesRequest))
          as GetTablesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTablesRequest create() => GetTablesRequest._();
  @$core.override
  GetTablesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTablesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTablesRequest>(create);
  static GetTablesRequest? _defaultInstance;
}

class GetTablesResponse extends $pb.GeneratedMessage {
  factory GetTablesResponse({
    $core.Iterable<Table>? tables,
  }) {
    final result = create();
    if (tables != null) result.tables.addAll(tables);
    return result;
  }

  GetTablesResponse._();

  factory GetTablesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTablesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTablesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..pPM<Table>(1, _omitFieldNames ? '' : 'tables', subBuilder: Table.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTablesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTablesResponse copyWith(void Function(GetTablesResponse) updates) =>
      super.copyWith((message) => updates(message as GetTablesResponse))
          as GetTablesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTablesResponse create() => GetTablesResponse._();
  @$core.override
  GetTablesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTablesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTablesResponse>(create);
  static GetTablesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Table> get tables => $_getList(0);
}

class Table extends $pb.GeneratedMessage {
  factory Table({
    $core.String? id,
    $core.String? hostId,
    $core.Iterable<Player>? players,
    $fixnum.Int64? smallBlind,
    $fixnum.Int64? bigBlind,
    $core.int? maxPlayers,
    $core.int? minPlayers,
    $core.int? currentPlayers,
    $fixnum.Int64? minBalance,
    $fixnum.Int64? buyIn,
    GamePhase? phase,
    $core.bool? gameStarted,
    $core.bool? allPlayersReady,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (hostId != null) result.hostId = hostId;
    if (players != null) result.players.addAll(players);
    if (smallBlind != null) result.smallBlind = smallBlind;
    if (bigBlind != null) result.bigBlind = bigBlind;
    if (maxPlayers != null) result.maxPlayers = maxPlayers;
    if (minPlayers != null) result.minPlayers = minPlayers;
    if (currentPlayers != null) result.currentPlayers = currentPlayers;
    if (minBalance != null) result.minBalance = minBalance;
    if (buyIn != null) result.buyIn = buyIn;
    if (phase != null) result.phase = phase;
    if (gameStarted != null) result.gameStarted = gameStarted;
    if (allPlayersReady != null) result.allPlayersReady = allPlayersReady;
    return result;
  }

  Table._();

  factory Table.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Table.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Table',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'hostId')
    ..pPM<Player>(3, _omitFieldNames ? '' : 'players',
        subBuilder: Player.create)
    ..aInt64(4, _omitFieldNames ? '' : 'smallBlind')
    ..aInt64(5, _omitFieldNames ? '' : 'bigBlind')
    ..aI(6, _omitFieldNames ? '' : 'maxPlayers')
    ..aI(7, _omitFieldNames ? '' : 'minPlayers')
    ..aI(8, _omitFieldNames ? '' : 'currentPlayers')
    ..aInt64(9, _omitFieldNames ? '' : 'minBalance')
    ..aInt64(10, _omitFieldNames ? '' : 'buyIn')
    ..aE<GamePhase>(11, _omitFieldNames ? '' : 'phase',
        enumValues: GamePhase.values)
    ..aOB(12, _omitFieldNames ? '' : 'gameStarted')
    ..aOB(13, _omitFieldNames ? '' : 'allPlayersReady')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Table clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Table copyWith(void Function(Table) updates) =>
      super.copyWith((message) => updates(message as Table)) as Table;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Table create() => Table._();
  @$core.override
  Table createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Table getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Table>(create);
  static Table? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get hostId => $_getSZ(1);
  @$pb.TagNumber(2)
  set hostId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHostId() => $_has(1);
  @$pb.TagNumber(2)
  void clearHostId() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<Player> get players => $_getList(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get smallBlind => $_getI64(3);
  @$pb.TagNumber(4)
  set smallBlind($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSmallBlind() => $_has(3);
  @$pb.TagNumber(4)
  void clearSmallBlind() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get bigBlind => $_getI64(4);
  @$pb.TagNumber(5)
  set bigBlind($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBigBlind() => $_has(4);
  @$pb.TagNumber(5)
  void clearBigBlind() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get maxPlayers => $_getIZ(5);
  @$pb.TagNumber(6)
  set maxPlayers($core.int value) => $_setSignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMaxPlayers() => $_has(5);
  @$pb.TagNumber(6)
  void clearMaxPlayers() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get minPlayers => $_getIZ(6);
  @$pb.TagNumber(7)
  set minPlayers($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMinPlayers() => $_has(6);
  @$pb.TagNumber(7)
  void clearMinPlayers() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get currentPlayers => $_getIZ(7);
  @$pb.TagNumber(8)
  set currentPlayers($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCurrentPlayers() => $_has(7);
  @$pb.TagNumber(8)
  void clearCurrentPlayers() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get minBalance => $_getI64(8);
  @$pb.TagNumber(9)
  set minBalance($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasMinBalance() => $_has(8);
  @$pb.TagNumber(9)
  void clearMinBalance() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get buyIn => $_getI64(9);
  @$pb.TagNumber(10)
  set buyIn($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasBuyIn() => $_has(9);
  @$pb.TagNumber(10)
  void clearBuyIn() => $_clearField(10);

  @$pb.TagNumber(11)
  GamePhase get phase => $_getN(10);
  @$pb.TagNumber(11)
  set phase(GamePhase value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasPhase() => $_has(10);
  @$pb.TagNumber(11)
  void clearPhase() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get gameStarted => $_getBF(11);
  @$pb.TagNumber(12)
  set gameStarted($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasGameStarted() => $_has(11);
  @$pb.TagNumber(12)
  void clearGameStarted() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.bool get allPlayersReady => $_getBF(12);
  @$pb.TagNumber(13)
  set allPlayersReady($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(13)
  $core.bool hasAllPlayersReady() => $_has(12);
  @$pb.TagNumber(13)
  void clearAllPlayersReady() => $_clearField(13);
}

class GetBalanceRequest extends $pb.GeneratedMessage {
  factory GetBalanceRequest({
    $core.String? playerId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    return result;
  }

  GetBalanceRequest._();

  factory GetBalanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetBalanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetBalanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetBalanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetBalanceRequest copyWith(void Function(GetBalanceRequest) updates) =>
      super.copyWith((message) => updates(message as GetBalanceRequest))
          as GetBalanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetBalanceRequest create() => GetBalanceRequest._();
  @$core.override
  GetBalanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetBalanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetBalanceRequest>(create);
  static GetBalanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);
}

class GetBalanceResponse extends $pb.GeneratedMessage {
  factory GetBalanceResponse({
    $fixnum.Int64? balance,
  }) {
    final result = create();
    if (balance != null) result.balance = balance;
    return result;
  }

  GetBalanceResponse._();

  factory GetBalanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetBalanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetBalanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'balance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetBalanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetBalanceResponse copyWith(void Function(GetBalanceResponse) updates) =>
      super.copyWith((message) => updates(message as GetBalanceResponse))
          as GetBalanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetBalanceResponse create() => GetBalanceResponse._();
  @$core.override
  GetBalanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetBalanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetBalanceResponse>(create);
  static GetBalanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get balance => $_getI64(0);
  @$pb.TagNumber(1)
  set balance($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBalance() => $_has(0);
  @$pb.TagNumber(1)
  void clearBalance() => $_clearField(1);
}

class UpdateBalanceRequest extends $pb.GeneratedMessage {
  factory UpdateBalanceRequest({
    $core.String? playerId,
    $fixnum.Int64? amount,
    $core.String? description,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (amount != null) result.amount = amount;
    if (description != null) result.description = description;
    return result;
  }

  UpdateBalanceRequest._();

  factory UpdateBalanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UpdateBalanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateBalanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aInt64(2, _omitFieldNames ? '' : 'amount')
    ..aOS(3, _omitFieldNames ? '' : 'description')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateBalanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateBalanceRequest copyWith(void Function(UpdateBalanceRequest) updates) =>
      super.copyWith((message) => updates(message as UpdateBalanceRequest))
          as UpdateBalanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateBalanceRequest create() => UpdateBalanceRequest._();
  @$core.override
  UpdateBalanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UpdateBalanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UpdateBalanceRequest>(create);
  static UpdateBalanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get amount => $_getI64(1);
  @$pb.TagNumber(2)
  set amount($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAmount() => $_has(1);
  @$pb.TagNumber(2)
  void clearAmount() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get description => $_getSZ(2);
  @$pb.TagNumber(3)
  set description($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDescription() => $_has(2);
  @$pb.TagNumber(3)
  void clearDescription() => $_clearField(3);
}

class UpdateBalanceResponse extends $pb.GeneratedMessage {
  factory UpdateBalanceResponse({
    $fixnum.Int64? newBalance,
    $core.String? message,
  }) {
    final result = create();
    if (newBalance != null) result.newBalance = newBalance;
    if (message != null) result.message = message;
    return result;
  }

  UpdateBalanceResponse._();

  factory UpdateBalanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UpdateBalanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateBalanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'newBalance')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateBalanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateBalanceResponse copyWith(
          void Function(UpdateBalanceResponse) updates) =>
      super.copyWith((message) => updates(message as UpdateBalanceResponse))
          as UpdateBalanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateBalanceResponse create() => UpdateBalanceResponse._();
  @$core.override
  UpdateBalanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UpdateBalanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UpdateBalanceResponse>(create);
  static UpdateBalanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get newBalance => $_getI64(0);
  @$pb.TagNumber(1)
  set newBalance($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNewBalance() => $_has(0);
  @$pb.TagNumber(1)
  void clearNewBalance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class ProcessTipRequest extends $pb.GeneratedMessage {
  factory ProcessTipRequest({
    $core.String? fromPlayerId,
    $core.String? toPlayerId,
    $fixnum.Int64? amount,
    $core.String? message,
  }) {
    final result = create();
    if (fromPlayerId != null) result.fromPlayerId = fromPlayerId;
    if (toPlayerId != null) result.toPlayerId = toPlayerId;
    if (amount != null) result.amount = amount;
    if (message != null) result.message = message;
    return result;
  }

  ProcessTipRequest._();

  factory ProcessTipRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProcessTipRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProcessTipRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'fromPlayerId')
    ..aOS(2, _omitFieldNames ? '' : 'toPlayerId')
    ..aInt64(3, _omitFieldNames ? '' : 'amount')
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProcessTipRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProcessTipRequest copyWith(void Function(ProcessTipRequest) updates) =>
      super.copyWith((message) => updates(message as ProcessTipRequest))
          as ProcessTipRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProcessTipRequest create() => ProcessTipRequest._();
  @$core.override
  ProcessTipRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProcessTipRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProcessTipRequest>(create);
  static ProcessTipRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get fromPlayerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set fromPlayerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFromPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFromPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get toPlayerId => $_getSZ(1);
  @$pb.TagNumber(2)
  set toPlayerId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasToPlayerId() => $_has(1);
  @$pb.TagNumber(2)
  void clearToPlayerId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get amount => $_getI64(2);
  @$pb.TagNumber(3)
  set amount($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAmount() => $_has(2);
  @$pb.TagNumber(3)
  void clearAmount() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get message => $_getSZ(3);
  @$pb.TagNumber(4)
  set message($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessage() => $_clearField(4);
}

class ProcessTipResponse extends $pb.GeneratedMessage {
  factory ProcessTipResponse({
    $core.bool? success,
    $core.String? message,
    $fixnum.Int64? newBalance,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    if (newBalance != null) result.newBalance = newBalance;
    return result;
  }

  ProcessTipResponse._();

  factory ProcessTipResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProcessTipResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProcessTipResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aInt64(3, _omitFieldNames ? '' : 'newBalance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProcessTipResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProcessTipResponse copyWith(void Function(ProcessTipResponse) updates) =>
      super.copyWith((message) => updates(message as ProcessTipResponse))
          as ProcessTipResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProcessTipResponse create() => ProcessTipResponse._();
  @$core.override
  ProcessTipResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProcessTipResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProcessTipResponse>(create);
  static ProcessTipResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get newBalance => $_getI64(2);
  @$pb.TagNumber(3)
  set newBalance($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNewBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewBalance() => $_clearField(3);
}

class StartNotificationStreamRequest extends $pb.GeneratedMessage {
  factory StartNotificationStreamRequest({
    $core.String? playerId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    return result;
  }

  StartNotificationStreamRequest._();

  factory StartNotificationStreamRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartNotificationStreamRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartNotificationStreamRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartNotificationStreamRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartNotificationStreamRequest copyWith(
          void Function(StartNotificationStreamRequest) updates) =>
      super.copyWith(
              (message) => updates(message as StartNotificationStreamRequest))
          as StartNotificationStreamRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartNotificationStreamRequest create() =>
      StartNotificationStreamRequest._();
  @$core.override
  StartNotificationStreamRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartNotificationStreamRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartNotificationStreamRequest>(create);
  static StartNotificationStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);
}

class Notification extends $pb.GeneratedMessage {
  factory Notification({
    NotificationType? type,
    $core.String? message,
    $core.String? tableId,
    $core.String? playerId,
    $fixnum.Int64? amount,
    $core.Iterable<Card>? cards,
    HandRank? handRank,
    $fixnum.Int64? newBalance,
    Table? table,
    $core.bool? ready,
    $core.bool? started,
    $core.bool? gameReadyToPlay,
    $core.int? countdown,
    $core.Iterable<Winner>? winners,
    Showdown? showdown,
    $core.String? winnerId,
    $core.int? winnerSeat,
    $core.String? matchId,
    $core.bool? isWinner,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (message != null) result.message = message;
    if (tableId != null) result.tableId = tableId;
    if (playerId != null) result.playerId = playerId;
    if (amount != null) result.amount = amount;
    if (cards != null) result.cards.addAll(cards);
    if (handRank != null) result.handRank = handRank;
    if (newBalance != null) result.newBalance = newBalance;
    if (table != null) result.table = table;
    if (ready != null) result.ready = ready;
    if (started != null) result.started = started;
    if (gameReadyToPlay != null) result.gameReadyToPlay = gameReadyToPlay;
    if (countdown != null) result.countdown = countdown;
    if (winners != null) result.winners.addAll(winners);
    if (showdown != null) result.showdown = showdown;
    if (winnerId != null) result.winnerId = winnerId;
    if (winnerSeat != null) result.winnerSeat = winnerSeat;
    if (matchId != null) result.matchId = matchId;
    if (isWinner != null) result.isWinner = isWinner;
    return result;
  }

  Notification._();

  factory Notification.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Notification.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Notification',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aE<NotificationType>(1, _omitFieldNames ? '' : 'type',
        enumValues: NotificationType.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOS(3, _omitFieldNames ? '' : 'tableId')
    ..aOS(4, _omitFieldNames ? '' : 'playerId')
    ..aInt64(5, _omitFieldNames ? '' : 'amount')
    ..pPM<Card>(6, _omitFieldNames ? '' : 'cards', subBuilder: Card.create)
    ..aE<HandRank>(7, _omitFieldNames ? '' : 'handRank',
        enumValues: HandRank.values)
    ..aInt64(8, _omitFieldNames ? '' : 'newBalance')
    ..aOM<Table>(9, _omitFieldNames ? '' : 'table', subBuilder: Table.create)
    ..aOB(10, _omitFieldNames ? '' : 'ready')
    ..aOB(11, _omitFieldNames ? '' : 'started')
    ..aOB(12, _omitFieldNames ? '' : 'gameReadyToPlay')
    ..aI(13, _omitFieldNames ? '' : 'countdown')
    ..pPM<Winner>(14, _omitFieldNames ? '' : 'winners',
        subBuilder: Winner.create)
    ..aOM<Showdown>(15, _omitFieldNames ? '' : 'showdown',
        subBuilder: Showdown.create)
    ..aOS(16, _omitFieldNames ? '' : 'winnerId')
    ..aI(17, _omitFieldNames ? '' : 'winnerSeat')
    ..aOS(18, _omitFieldNames ? '' : 'matchId')
    ..aOB(19, _omitFieldNames ? '' : 'isWinner')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Notification clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Notification copyWith(void Function(Notification) updates) =>
      super.copyWith((message) => updates(message as Notification))
          as Notification;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Notification create() => Notification._();
  @$core.override
  Notification createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Notification getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Notification>(create);
  static Notification? _defaultInstance;

  @$pb.TagNumber(1)
  NotificationType get type => $_getN(0);
  @$pb.TagNumber(1)
  set type(NotificationType value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get tableId => $_getSZ(2);
  @$pb.TagNumber(3)
  set tableId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTableId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTableId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get playerId => $_getSZ(3);
  @$pb.TagNumber(4)
  set playerId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlayerId() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlayerId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get amount => $_getI64(4);
  @$pb.TagNumber(5)
  set amount($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAmount() => $_has(4);
  @$pb.TagNumber(5)
  void clearAmount() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<Card> get cards => $_getList(5);

  @$pb.TagNumber(7)
  HandRank get handRank => $_getN(6);
  @$pb.TagNumber(7)
  set handRank(HandRank value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasHandRank() => $_has(6);
  @$pb.TagNumber(7)
  void clearHandRank() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get newBalance => $_getI64(7);
  @$pb.TagNumber(8)
  set newBalance($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasNewBalance() => $_has(7);
  @$pb.TagNumber(8)
  void clearNewBalance() => $_clearField(8);

  @$pb.TagNumber(9)
  Table get table => $_getN(8);
  @$pb.TagNumber(9)
  set table(Table value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasTable() => $_has(8);
  @$pb.TagNumber(9)
  void clearTable() => $_clearField(9);
  @$pb.TagNumber(9)
  Table ensureTable() => $_ensure(8);

  @$pb.TagNumber(10)
  $core.bool get ready => $_getBF(9);
  @$pb.TagNumber(10)
  set ready($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasReady() => $_has(9);
  @$pb.TagNumber(10)
  void clearReady() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.bool get started => $_getBF(10);
  @$pb.TagNumber(11)
  set started($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(11)
  $core.bool hasStarted() => $_has(10);
  @$pb.TagNumber(11)
  void clearStarted() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get gameReadyToPlay => $_getBF(11);
  @$pb.TagNumber(12)
  set gameReadyToPlay($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasGameReadyToPlay() => $_has(11);
  @$pb.TagNumber(12)
  void clearGameReadyToPlay() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.int get countdown => $_getIZ(12);
  @$pb.TagNumber(13)
  set countdown($core.int value) => $_setSignedInt32(12, value);
  @$pb.TagNumber(13)
  $core.bool hasCountdown() => $_has(12);
  @$pb.TagNumber(13)
  void clearCountdown() => $_clearField(13);

  @$pb.TagNumber(14)
  $pb.PbList<Winner> get winners => $_getList(13);

  @$pb.TagNumber(15)
  Showdown get showdown => $_getN(14);
  @$pb.TagNumber(15)
  set showdown(Showdown value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasShowdown() => $_has(14);
  @$pb.TagNumber(15)
  void clearShowdown() => $_clearField(15);
  @$pb.TagNumber(15)
  Showdown ensureShowdown() => $_ensure(14);

  /// Settlement fields for GAME_ENDED notifications
  @$pb.TagNumber(16)
  $core.String get winnerId => $_getSZ(15);
  @$pb.TagNumber(16)
  set winnerId($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasWinnerId() => $_has(15);
  @$pb.TagNumber(16)
  void clearWinnerId() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.int get winnerSeat => $_getIZ(16);
  @$pb.TagNumber(17)
  set winnerSeat($core.int value) => $_setSignedInt32(16, value);
  @$pb.TagNumber(17)
  $core.bool hasWinnerSeat() => $_has(16);
  @$pb.TagNumber(17)
  void clearWinnerSeat() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.String get matchId => $_getSZ(17);
  @$pb.TagNumber(18)
  set matchId($core.String value) => $_setString(17, value);
  @$pb.TagNumber(18)
  $core.bool hasMatchId() => $_has(17);
  @$pb.TagNumber(18)
  void clearMatchId() => $_clearField(18);

  @$pb.TagNumber(19)
  $core.bool get isWinner => $_getBF(18);
  @$pb.TagNumber(19)
  set isWinner($core.bool value) => $_setBool(18, value);
  @$pb.TagNumber(19)
  $core.bool hasIsWinner() => $_has(18);
  @$pb.TagNumber(19)
  void clearIsWinner() => $_clearField(19);
}

class Showdown extends $pb.GeneratedMessage {
  factory Showdown({
    $core.Iterable<Winner>? winners,
    $fixnum.Int64? pot,
  }) {
    final result = create();
    if (winners != null) result.winners.addAll(winners);
    if (pot != null) result.pot = pot;
    return result;
  }

  Showdown._();

  factory Showdown.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Showdown.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Showdown',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..pPM<Winner>(1, _omitFieldNames ? '' : 'winners',
        subBuilder: Winner.create)
    ..aInt64(2, _omitFieldNames ? '' : 'pot')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Showdown clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Showdown copyWith(void Function(Showdown) updates) =>
      super.copyWith((message) => updates(message as Showdown)) as Showdown;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Showdown create() => Showdown._();
  @$core.override
  Showdown createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Showdown getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Showdown>(create);
  static Showdown? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Winner> get winners => $_getList(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get pot => $_getI64(1);
  @$pb.TagNumber(2)
  set pot($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPot() => $_has(1);
  @$pb.TagNumber(2)
  void clearPot() => $_clearField(2);
}

/// Common Messages
class Player extends $pb.GeneratedMessage {
  factory Player({
    $core.String? id,
    $core.String? name,
    $fixnum.Int64? balance,
    $core.Iterable<Card>? hand,
    $fixnum.Int64? currentBet,
    $core.bool? folded,
    $core.bool? isTurn,
    $core.bool? isAllIn,
    $core.bool? isDealer,
    $core.bool? isReady,
    $core.String? handDescription,
    PlayerState? playerState,
    $core.bool? isSmallBlind,
    $core.bool? isBigBlind,
    $core.bool? isDisconnected,
    $core.String? escrowId,
    $core.bool? escrowReady,
    $core.int? tableSeat,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (balance != null) result.balance = balance;
    if (hand != null) result.hand.addAll(hand);
    if (currentBet != null) result.currentBet = currentBet;
    if (folded != null) result.folded = folded;
    if (isTurn != null) result.isTurn = isTurn;
    if (isAllIn != null) result.isAllIn = isAllIn;
    if (isDealer != null) result.isDealer = isDealer;
    if (isReady != null) result.isReady = isReady;
    if (handDescription != null) result.handDescription = handDescription;
    if (playerState != null) result.playerState = playerState;
    if (isSmallBlind != null) result.isSmallBlind = isSmallBlind;
    if (isBigBlind != null) result.isBigBlind = isBigBlind;
    if (isDisconnected != null) result.isDisconnected = isDisconnected;
    if (escrowId != null) result.escrowId = escrowId;
    if (escrowReady != null) result.escrowReady = escrowReady;
    if (tableSeat != null) result.tableSeat = tableSeat;
    return result;
  }

  Player._();

  factory Player.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Player.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Player',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aInt64(3, _omitFieldNames ? '' : 'balance')
    ..pPM<Card>(4, _omitFieldNames ? '' : 'hand', subBuilder: Card.create)
    ..aInt64(5, _omitFieldNames ? '' : 'currentBet')
    ..aOB(6, _omitFieldNames ? '' : 'folded')
    ..aOB(7, _omitFieldNames ? '' : 'isTurn')
    ..aOB(8, _omitFieldNames ? '' : 'isAllIn')
    ..aOB(9, _omitFieldNames ? '' : 'isDealer')
    ..aOB(10, _omitFieldNames ? '' : 'isReady')
    ..aOS(11, _omitFieldNames ? '' : 'handDescription')
    ..aE<PlayerState>(12, _omitFieldNames ? '' : 'playerState',
        enumValues: PlayerState.values)
    ..aOB(13, _omitFieldNames ? '' : 'isSmallBlind')
    ..aOB(14, _omitFieldNames ? '' : 'isBigBlind')
    ..aOB(15, _omitFieldNames ? '' : 'isDisconnected')
    ..aOS(16, _omitFieldNames ? '' : 'escrowId')
    ..aOB(17, _omitFieldNames ? '' : 'escrowReady')
    ..aI(18, _omitFieldNames ? '' : 'tableSeat')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Player clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Player copyWith(void Function(Player) updates) =>
      super.copyWith((message) => updates(message as Player)) as Player;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Player create() => Player._();
  @$core.override
  Player createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Player getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Player>(create);
  static Player? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get balance => $_getI64(2);
  @$pb.TagNumber(3)
  set balance($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearBalance() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<Card> get hand => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get currentBet => $_getI64(4);
  @$pb.TagNumber(5)
  set currentBet($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCurrentBet() => $_has(4);
  @$pb.TagNumber(5)
  void clearCurrentBet() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get folded => $_getBF(5);
  @$pb.TagNumber(6)
  set folded($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFolded() => $_has(5);
  @$pb.TagNumber(6)
  void clearFolded() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get isTurn => $_getBF(6);
  @$pb.TagNumber(7)
  set isTurn($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasIsTurn() => $_has(6);
  @$pb.TagNumber(7)
  void clearIsTurn() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get isAllIn => $_getBF(7);
  @$pb.TagNumber(8)
  set isAllIn($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasIsAllIn() => $_has(7);
  @$pb.TagNumber(8)
  void clearIsAllIn() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get isDealer => $_getBF(8);
  @$pb.TagNumber(9)
  set isDealer($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIsDealer() => $_has(8);
  @$pb.TagNumber(9)
  void clearIsDealer() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get isReady => $_getBF(9);
  @$pb.TagNumber(10)
  set isReady($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasIsReady() => $_has(9);
  @$pb.TagNumber(10)
  void clearIsReady() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get handDescription => $_getSZ(10);
  @$pb.TagNumber(11)
  set handDescription($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasHandDescription() => $_has(10);
  @$pb.TagNumber(11)
  void clearHandDescription() => $_clearField(11);

  @$pb.TagNumber(12)
  PlayerState get playerState => $_getN(11);
  @$pb.TagNumber(12)
  set playerState(PlayerState value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasPlayerState() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlayerState() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.bool get isSmallBlind => $_getBF(12);
  @$pb.TagNumber(13)
  set isSmallBlind($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(13)
  $core.bool hasIsSmallBlind() => $_has(12);
  @$pb.TagNumber(13)
  void clearIsSmallBlind() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.bool get isBigBlind => $_getBF(13);
  @$pb.TagNumber(14)
  set isBigBlind($core.bool value) => $_setBool(13, value);
  @$pb.TagNumber(14)
  $core.bool hasIsBigBlind() => $_has(13);
  @$pb.TagNumber(14)
  void clearIsBigBlind() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get isDisconnected => $_getBF(14);
  @$pb.TagNumber(15)
  set isDisconnected($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(15)
  $core.bool hasIsDisconnected() => $_has(14);
  @$pb.TagNumber(15)
  void clearIsDisconnected() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get escrowId => $_getSZ(15);
  @$pb.TagNumber(16)
  set escrowId($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasEscrowId() => $_has(15);
  @$pb.TagNumber(16)
  void clearEscrowId() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.bool get escrowReady => $_getBF(16);
  @$pb.TagNumber(17)
  set escrowReady($core.bool value) => $_setBool(16, value);
  @$pb.TagNumber(17)
  $core.bool hasEscrowReady() => $_has(16);
  @$pb.TagNumber(17)
  void clearEscrowReady() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.int get tableSeat => $_getIZ(17);
  @$pb.TagNumber(18)
  set tableSeat($core.int value) => $_setSignedInt32(17, value);
  @$pb.TagNumber(18)
  $core.bool hasTableSeat() => $_has(17);
  @$pb.TagNumber(18)
  void clearTableSeat() => $_clearField(18);
}

class Card extends $pb.GeneratedMessage {
  factory Card({
    $core.String? suit,
    $core.String? value,
  }) {
    final result = create();
    if (suit != null) result.suit = suit;
    if (value != null) result.value = value;
    return result;
  }

  Card._();

  factory Card.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Card.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Card',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'suit')
    ..aOS(2, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Card clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Card copyWith(void Function(Card) updates) =>
      super.copyWith((message) => updates(message as Card)) as Card;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Card create() => Card._();
  @$core.override
  Card createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Card getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Card>(create);
  static Card? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get suit => $_getSZ(0);
  @$pb.TagNumber(1)
  set suit($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuit() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuit() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get value => $_getSZ(1);
  @$pb.TagNumber(2)
  set value($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(2)
  void clearValue() => $_clearField(2);
}

class SetPlayerReadyRequest extends $pb.GeneratedMessage {
  factory SetPlayerReadyRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  SetPlayerReadyRequest._();

  factory SetPlayerReadyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetPlayerReadyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetPlayerReadyRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerReadyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerReadyRequest copyWith(
          void Function(SetPlayerReadyRequest) updates) =>
      super.copyWith((message) => updates(message as SetPlayerReadyRequest))
          as SetPlayerReadyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyRequest create() => SetPlayerReadyRequest._();
  @$core.override
  SetPlayerReadyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetPlayerReadyRequest>(create);
  static SetPlayerReadyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class SetPlayerReadyResponse extends $pb.GeneratedMessage {
  factory SetPlayerReadyResponse({
    $core.bool? success,
    $core.String? message,
    $core.bool? allPlayersReady,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    if (allPlayersReady != null) result.allPlayersReady = allPlayersReady;
    return result;
  }

  SetPlayerReadyResponse._();

  factory SetPlayerReadyResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetPlayerReadyResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetPlayerReadyResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOB(3, _omitFieldNames ? '' : 'allPlayersReady')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerReadyResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerReadyResponse copyWith(
          void Function(SetPlayerReadyResponse) updates) =>
      super.copyWith((message) => updates(message as SetPlayerReadyResponse))
          as SetPlayerReadyResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyResponse create() => SetPlayerReadyResponse._();
  @$core.override
  SetPlayerReadyResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetPlayerReadyResponse>(create);
  static SetPlayerReadyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get allPlayersReady => $_getBF(2);
  @$pb.TagNumber(3)
  set allPlayersReady($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAllPlayersReady() => $_has(2);
  @$pb.TagNumber(3)
  void clearAllPlayersReady() => $_clearField(3);
}

class SetPlayerUnreadyRequest extends $pb.GeneratedMessage {
  factory SetPlayerUnreadyRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  SetPlayerUnreadyRequest._();

  factory SetPlayerUnreadyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetPlayerUnreadyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetPlayerUnreadyRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerUnreadyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerUnreadyRequest copyWith(
          void Function(SetPlayerUnreadyRequest) updates) =>
      super.copyWith((message) => updates(message as SetPlayerUnreadyRequest))
          as SetPlayerUnreadyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyRequest create() => SetPlayerUnreadyRequest._();
  @$core.override
  SetPlayerUnreadyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetPlayerUnreadyRequest>(create);
  static SetPlayerUnreadyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class SetPlayerUnreadyResponse extends $pb.GeneratedMessage {
  factory SetPlayerUnreadyResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  SetPlayerUnreadyResponse._();

  factory SetPlayerUnreadyResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetPlayerUnreadyResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetPlayerUnreadyResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerUnreadyResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetPlayerUnreadyResponse copyWith(
          void Function(SetPlayerUnreadyResponse) updates) =>
      super.copyWith((message) => updates(message as SetPlayerUnreadyResponse))
          as SetPlayerUnreadyResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyResponse create() => SetPlayerUnreadyResponse._();
  @$core.override
  SetPlayerUnreadyResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetPlayerUnreadyResponse>(create);
  static SetPlayerUnreadyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class GetPlayerCurrentTableRequest extends $pb.GeneratedMessage {
  factory GetPlayerCurrentTableRequest({
    $core.String? playerId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    return result;
  }

  GetPlayerCurrentTableRequest._();

  factory GetPlayerCurrentTableRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetPlayerCurrentTableRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPlayerCurrentTableRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPlayerCurrentTableRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPlayerCurrentTableRequest copyWith(
          void Function(GetPlayerCurrentTableRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetPlayerCurrentTableRequest))
          as GetPlayerCurrentTableRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableRequest create() =>
      GetPlayerCurrentTableRequest._();
  @$core.override
  GetPlayerCurrentTableRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetPlayerCurrentTableRequest>(create);
  static GetPlayerCurrentTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);
}

class GetPlayerCurrentTableResponse extends $pb.GeneratedMessage {
  factory GetPlayerCurrentTableResponse({
    $core.String? tableId,
  }) {
    final result = create();
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  GetPlayerCurrentTableResponse._();

  factory GetPlayerCurrentTableResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetPlayerCurrentTableResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPlayerCurrentTableResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPlayerCurrentTableResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPlayerCurrentTableResponse copyWith(
          void Function(GetPlayerCurrentTableResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetPlayerCurrentTableResponse))
          as GetPlayerCurrentTableResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableResponse create() =>
      GetPlayerCurrentTableResponse._();
  @$core.override
  GetPlayerCurrentTableResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetPlayerCurrentTableResponse>(create);
  static GetPlayerCurrentTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => $_clearField(1);
}

class ShowCardsRequest extends $pb.GeneratedMessage {
  factory ShowCardsRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  ShowCardsRequest._();

  factory ShowCardsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShowCardsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShowCardsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowCardsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowCardsRequest copyWith(void Function(ShowCardsRequest) updates) =>
      super.copyWith((message) => updates(message as ShowCardsRequest))
          as ShowCardsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShowCardsRequest create() => ShowCardsRequest._();
  @$core.override
  ShowCardsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShowCardsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShowCardsRequest>(create);
  static ShowCardsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class ShowCardsResponse extends $pb.GeneratedMessage {
  factory ShowCardsResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  ShowCardsResponse._();

  factory ShowCardsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShowCardsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShowCardsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowCardsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowCardsResponse copyWith(void Function(ShowCardsResponse) updates) =>
      super.copyWith((message) => updates(message as ShowCardsResponse))
          as ShowCardsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShowCardsResponse create() => ShowCardsResponse._();
  @$core.override
  ShowCardsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShowCardsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShowCardsResponse>(create);
  static ShowCardsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class HideCardsRequest extends $pb.GeneratedMessage {
  factory HideCardsRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final result = create();
    if (playerId != null) result.playerId = playerId;
    if (tableId != null) result.tableId = tableId;
    return result;
  }

  HideCardsRequest._();

  factory HideCardsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HideCardsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HideCardsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HideCardsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HideCardsRequest copyWith(void Function(HideCardsRequest) updates) =>
      super.copyWith((message) => updates(message as HideCardsRequest))
          as HideCardsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HideCardsRequest create() => HideCardsRequest._();
  @$core.override
  HideCardsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HideCardsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HideCardsRequest>(create);
  static HideCardsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);
}

class HideCardsResponse extends $pb.GeneratedMessage {
  factory HideCardsResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final result = create();
    if (success != null) result.success = success;
    if (message != null) result.message = message;
    return result;
  }

  HideCardsResponse._();

  factory HideCardsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HideCardsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HideCardsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HideCardsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HideCardsResponse copyWith(void Function(HideCardsResponse) updates) =>
      super.copyWith((message) => updates(message as HideCardsResponse))
          as HideCardsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HideCardsResponse create() => HideCardsResponse._();
  @$core.override
  HideCardsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HideCardsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HideCardsResponse>(create);
  static HideCardsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

/// Auth Messages
class RegisterRequest extends $pb.GeneratedMessage {
  factory RegisterRequest({
    $core.String? nickname,
    $core.String? userId,
  }) {
    final result = create();
    if (nickname != null) result.nickname = nickname;
    if (userId != null) result.userId = userId;
    return result;
  }

  RegisterRequest._();

  factory RegisterRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'nickname')
    ..aOS(2, _omitFieldNames ? '' : 'userId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterRequest copyWith(void Function(RegisterRequest) updates) =>
      super.copyWith((message) => updates(message as RegisterRequest))
          as RegisterRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterRequest create() => RegisterRequest._();
  @$core.override
  RegisterRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterRequest>(create);
  static RegisterRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get nickname => $_getSZ(0);
  @$pb.TagNumber(1)
  set nickname($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNickname() => $_has(0);
  @$pb.TagNumber(1)
  void clearNickname() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get userId => $_getSZ(1);
  @$pb.TagNumber(2)
  set userId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasUserId() => $_has(1);
  @$pb.TagNumber(2)
  void clearUserId() => $_clearField(2);
}

class RegisterResponse extends $pb.GeneratedMessage {
  factory RegisterResponse({
    $core.bool? ok,
    $core.String? error,
  }) {
    final result = create();
    if (ok != null) result.ok = ok;
    if (error != null) result.error = error;
    return result;
  }

  RegisterResponse._();

  factory RegisterResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'ok')
    ..aOS(2, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterResponse copyWith(void Function(RegisterResponse) updates) =>
      super.copyWith((message) => updates(message as RegisterResponse))
          as RegisterResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterResponse create() => RegisterResponse._();
  @$core.override
  RegisterResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterResponse>(create);
  static RegisterResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get ok => $_getBF(0);
  @$pb.TagNumber(1)
  set ok($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOk() => $_has(0);
  @$pb.TagNumber(1)
  void clearOk() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get error => $_getSZ(1);
  @$pb.TagNumber(2)
  set error($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
}

class LoginRequest extends $pb.GeneratedMessage {
  factory LoginRequest({
    $core.String? nickname,
    $core.String? userId,
    $core.String? address,
    $core.String? signature,
    $core.String? code,
  }) {
    final result = create();
    if (nickname != null) result.nickname = nickname;
    if (userId != null) result.userId = userId;
    if (address != null) result.address = address;
    if (signature != null) result.signature = signature;
    if (code != null) result.code = code;
    return result;
  }

  LoginRequest._();

  factory LoginRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LoginRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LoginRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'nickname')
    ..aOS(2, _omitFieldNames ? '' : 'userId')
    ..aOS(3, _omitFieldNames ? '' : 'address')
    ..aOS(4, _omitFieldNames ? '' : 'signature')
    ..aOS(5, _omitFieldNames ? '' : 'code')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginRequest copyWith(void Function(LoginRequest) updates) =>
      super.copyWith((message) => updates(message as LoginRequest))
          as LoginRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LoginRequest create() => LoginRequest._();
  @$core.override
  LoginRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LoginRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LoginRequest>(create);
  static LoginRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get nickname => $_getSZ(0);
  @$pb.TagNumber(1)
  set nickname($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNickname() => $_has(0);
  @$pb.TagNumber(1)
  void clearNickname() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get userId => $_getSZ(1);
  @$pb.TagNumber(2)
  set userId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasUserId() => $_has(1);
  @$pb.TagNumber(2)
  void clearUserId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get address => $_getSZ(2);
  @$pb.TagNumber(3)
  set address($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAddress() => $_has(2);
  @$pb.TagNumber(3)
  void clearAddress() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get signature => $_getSZ(3);
  @$pb.TagNumber(4)
  set signature($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignature() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignature() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get code => $_getSZ(4);
  @$pb.TagNumber(5)
  set code($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCode() => $_has(4);
  @$pb.TagNumber(5)
  void clearCode() => $_clearField(5);
}

class RequestLoginCodeRequest extends $pb.GeneratedMessage {
  factory RequestLoginCodeRequest({
    $core.String? userId,
  }) {
    final result = create();
    if (userId != null) result.userId = userId;
    return result;
  }

  RequestLoginCodeRequest._();

  factory RequestLoginCodeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestLoginCodeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestLoginCodeRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'userId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestLoginCodeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestLoginCodeRequest copyWith(
          void Function(RequestLoginCodeRequest) updates) =>
      super.copyWith((message) => updates(message as RequestLoginCodeRequest))
          as RequestLoginCodeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestLoginCodeRequest create() => RequestLoginCodeRequest._();
  @$core.override
  RequestLoginCodeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestLoginCodeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestLoginCodeRequest>(create);
  static RequestLoginCodeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get userId => $_getSZ(0);
  @$pb.TagNumber(1)
  set userId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUserId() => $_has(0);
  @$pb.TagNumber(1)
  void clearUserId() => $_clearField(1);
}

class RequestLoginCodeResponse extends $pb.GeneratedMessage {
  factory RequestLoginCodeResponse({
    $core.String? code,
    $fixnum.Int64? ttlSec,
    $core.String? addressHint,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (ttlSec != null) result.ttlSec = ttlSec;
    if (addressHint != null) result.addressHint = addressHint;
    return result;
  }

  RequestLoginCodeResponse._();

  factory RequestLoginCodeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestLoginCodeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestLoginCodeResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'code')
    ..aInt64(2, _omitFieldNames ? '' : 'ttlSec')
    ..aOS(3, _omitFieldNames ? '' : 'addressHint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestLoginCodeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestLoginCodeResponse copyWith(
          void Function(RequestLoginCodeResponse) updates) =>
      super.copyWith((message) => updates(message as RequestLoginCodeResponse))
          as RequestLoginCodeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestLoginCodeResponse create() => RequestLoginCodeResponse._();
  @$core.override
  RequestLoginCodeResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestLoginCodeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestLoginCodeResponse>(create);
  static RequestLoginCodeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get code => $_getSZ(0);
  @$pb.TagNumber(1)
  set code($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get ttlSec => $_getI64(1);
  @$pb.TagNumber(2)
  set ttlSec($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTtlSec() => $_has(1);
  @$pb.TagNumber(2)
  void clearTtlSec() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get addressHint => $_getSZ(2);
  @$pb.TagNumber(3)
  set addressHint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAddressHint() => $_has(2);
  @$pb.TagNumber(3)
  void clearAddressHint() => $_clearField(3);
}

class LoginResponse extends $pb.GeneratedMessage {
  factory LoginResponse({
    $core.bool? ok,
    $core.String? error,
    $core.String? token,
    $core.String? userId,
    $core.String? nickname,
    $core.String? payoutAddress,
  }) {
    final result = create();
    if (ok != null) result.ok = ok;
    if (error != null) result.error = error;
    if (token != null) result.token = token;
    if (userId != null) result.userId = userId;
    if (nickname != null) result.nickname = nickname;
    if (payoutAddress != null) result.payoutAddress = payoutAddress;
    return result;
  }

  LoginResponse._();

  factory LoginResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LoginResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LoginResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'ok')
    ..aOS(2, _omitFieldNames ? '' : 'error')
    ..aOS(3, _omitFieldNames ? '' : 'token')
    ..aOS(4, _omitFieldNames ? '' : 'userId')
    ..aOS(5, _omitFieldNames ? '' : 'nickname')
    ..aOS(6, _omitFieldNames ? '' : 'payoutAddress')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginResponse copyWith(void Function(LoginResponse) updates) =>
      super.copyWith((message) => updates(message as LoginResponse))
          as LoginResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LoginResponse create() => LoginResponse._();
  @$core.override
  LoginResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LoginResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LoginResponse>(create);
  static LoginResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get ok => $_getBF(0);
  @$pb.TagNumber(1)
  set ok($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOk() => $_has(0);
  @$pb.TagNumber(1)
  void clearOk() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get error => $_getSZ(1);
  @$pb.TagNumber(2)
  set error($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get token => $_getSZ(2);
  @$pb.TagNumber(3)
  set token($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasToken() => $_has(2);
  @$pb.TagNumber(3)
  void clearToken() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get userId => $_getSZ(3);
  @$pb.TagNumber(4)
  set userId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUserId() => $_has(3);
  @$pb.TagNumber(4)
  void clearUserId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get nickname => $_getSZ(4);
  @$pb.TagNumber(5)
  set nickname($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasNickname() => $_has(4);
  @$pb.TagNumber(5)
  void clearNickname() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get payoutAddress => $_getSZ(5);
  @$pb.TagNumber(6)
  set payoutAddress($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPayoutAddress() => $_has(5);
  @$pb.TagNumber(6)
  void clearPayoutAddress() => $_clearField(6);
}

class LogoutRequest extends $pb.GeneratedMessage {
  factory LogoutRequest({
    $core.String? token,
  }) {
    final result = create();
    if (token != null) result.token = token;
    return result;
  }

  LogoutRequest._();

  factory LogoutRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LogoutRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LogoutRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'token')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutRequest copyWith(void Function(LogoutRequest) updates) =>
      super.copyWith((message) => updates(message as LogoutRequest))
          as LogoutRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LogoutRequest create() => LogoutRequest._();
  @$core.override
  LogoutRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LogoutRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LogoutRequest>(create);
  static LogoutRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get token => $_getSZ(0);
  @$pb.TagNumber(1)
  set token($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearToken() => $_clearField(1);
}

class LogoutResponse extends $pb.GeneratedMessage {
  factory LogoutResponse({
    $core.bool? ok,
  }) {
    final result = create();
    if (ok != null) result.ok = ok;
    return result;
  }

  LogoutResponse._();

  factory LogoutResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LogoutResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LogoutResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'ok')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutResponse copyWith(void Function(LogoutResponse) updates) =>
      super.copyWith((message) => updates(message as LogoutResponse))
          as LogoutResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LogoutResponse create() => LogoutResponse._();
  @$core.override
  LogoutResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LogoutResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LogoutResponse>(create);
  static LogoutResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get ok => $_getBF(0);
  @$pb.TagNumber(1)
  set ok($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOk() => $_has(0);
  @$pb.TagNumber(1)
  void clearOk() => $_clearField(1);
}

class GetUserInfoRequest extends $pb.GeneratedMessage {
  factory GetUserInfoRequest({
    $core.String? token,
  }) {
    final result = create();
    if (token != null) result.token = token;
    return result;
  }

  GetUserInfoRequest._();

  factory GetUserInfoRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserInfoRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserInfoRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'token')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoRequest copyWith(void Function(GetUserInfoRequest) updates) =>
      super.copyWith((message) => updates(message as GetUserInfoRequest))
          as GetUserInfoRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserInfoRequest create() => GetUserInfoRequest._();
  @$core.override
  GetUserInfoRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserInfoRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserInfoRequest>(create);
  static GetUserInfoRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get token => $_getSZ(0);
  @$pb.TagNumber(1)
  set token($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearToken() => $_clearField(1);
}

class GetUserInfoResponse extends $pb.GeneratedMessage {
  factory GetUserInfoResponse({
    $core.String? userId,
    $core.String? nickname,
    $fixnum.Int64? created,
    $fixnum.Int64? lastLogin,
  }) {
    final result = create();
    if (userId != null) result.userId = userId;
    if (nickname != null) result.nickname = nickname;
    if (created != null) result.created = created;
    if (lastLogin != null) result.lastLogin = lastLogin;
    return result;
  }

  GetUserInfoResponse._();

  factory GetUserInfoResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserInfoResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserInfoResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'userId')
    ..aOS(2, _omitFieldNames ? '' : 'nickname')
    ..aInt64(3, _omitFieldNames ? '' : 'created')
    ..aInt64(4, _omitFieldNames ? '' : 'lastLogin')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoResponse copyWith(void Function(GetUserInfoResponse) updates) =>
      super.copyWith((message) => updates(message as GetUserInfoResponse))
          as GetUserInfoResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserInfoResponse create() => GetUserInfoResponse._();
  @$core.override
  GetUserInfoResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserInfoResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserInfoResponse>(create);
  static GetUserInfoResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get userId => $_getSZ(0);
  @$pb.TagNumber(1)
  set userId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUserId() => $_has(0);
  @$pb.TagNumber(1)
  void clearUserId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get nickname => $_getSZ(1);
  @$pb.TagNumber(2)
  set nickname($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNickname() => $_has(1);
  @$pb.TagNumber(2)
  void clearNickname() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get created => $_getI64(2);
  @$pb.TagNumber(3)
  set created($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCreated() => $_has(2);
  @$pb.TagNumber(3)
  void clearCreated() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get lastLogin => $_getI64(3);
  @$pb.TagNumber(4)
  set lastLogin($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLastLogin() => $_has(3);
  @$pb.TagNumber(4)
  void clearLastLogin() => $_clearField(4);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
