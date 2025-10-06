//
//  Generated code. Do not modify.
//  source: poker.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'poker.pbenum.dart';

export 'poker.pbenum.dart';

/// Game Messages
class StartGameStreamRequest extends $pb.GeneratedMessage {
  factory StartGameStreamRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  StartGameStreamRequest._() : super();
  factory StartGameStreamRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory StartGameStreamRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'StartGameStreamRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  StartGameStreamRequest clone() => StartGameStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  StartGameStreamRequest copyWith(void Function(StartGameStreamRequest) updates) => super.copyWith((message) => updates(message as StartGameStreamRequest)) as StartGameStreamRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartGameStreamRequest create() => StartGameStreamRequest._();
  StartGameStreamRequest createEmptyInstance() => create();
  static $pb.PbList<StartGameStreamRequest> createRepeated() => $pb.PbList<StartGameStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static StartGameStreamRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StartGameStreamRequest>(create);
  static StartGameStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
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
  }) {
    final $result = create();
    if (tableId != null) {
      $result.tableId = tableId;
    }
    if (phase != null) {
      $result.phase = phase;
    }
    if (players != null) {
      $result.players.addAll(players);
    }
    if (communityCards != null) {
      $result.communityCards.addAll(communityCards);
    }
    if (pot != null) {
      $result.pot = pot;
    }
    if (currentBet != null) {
      $result.currentBet = currentBet;
    }
    if (currentPlayer != null) {
      $result.currentPlayer = currentPlayer;
    }
    if (minRaise != null) {
      $result.minRaise = minRaise;
    }
    if (maxRaise != null) {
      $result.maxRaise = maxRaise;
    }
    if (gameStarted != null) {
      $result.gameStarted = gameStarted;
    }
    if (playersRequired != null) {
      $result.playersRequired = playersRequired;
    }
    if (playersJoined != null) {
      $result.playersJoined = playersJoined;
    }
    if (phaseName != null) {
      $result.phaseName = phaseName;
    }
    return $result;
  }
  GameUpdate._() : super();
  factory GameUpdate.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GameUpdate.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GameUpdate', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..e<GamePhase>(2, _omitFieldNames ? '' : 'phase', $pb.PbFieldType.OE, defaultOrMaker: GamePhase.WAITING, valueOf: GamePhase.valueOf, enumValues: GamePhase.values)
    ..pc<Player>(3, _omitFieldNames ? '' : 'players', $pb.PbFieldType.PM, subBuilder: Player.create)
    ..pc<Card>(4, _omitFieldNames ? '' : 'communityCards', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..aInt64(5, _omitFieldNames ? '' : 'pot')
    ..aInt64(6, _omitFieldNames ? '' : 'currentBet')
    ..aOS(7, _omitFieldNames ? '' : 'currentPlayer')
    ..aInt64(8, _omitFieldNames ? '' : 'minRaise')
    ..aInt64(9, _omitFieldNames ? '' : 'maxRaise')
    ..aOB(10, _omitFieldNames ? '' : 'gameStarted')
    ..a<$core.int>(11, _omitFieldNames ? '' : 'playersRequired', $pb.PbFieldType.O3)
    ..a<$core.int>(12, _omitFieldNames ? '' : 'playersJoined', $pb.PbFieldType.O3)
    ..aOS(13, _omitFieldNames ? '' : 'phaseName')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GameUpdate clone() => GameUpdate()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GameUpdate copyWith(void Function(GameUpdate) updates) => super.copyWith((message) => updates(message as GameUpdate)) as GameUpdate;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GameUpdate create() => GameUpdate._();
  GameUpdate createEmptyInstance() => create();
  static $pb.PbList<GameUpdate> createRepeated() => $pb.PbList<GameUpdate>();
  @$core.pragma('dart2js:noInline')
  static GameUpdate getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GameUpdate>(create);
  static GameUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => clearField(1);

  @$pb.TagNumber(2)
  GamePhase get phase => $_getN(1);
  @$pb.TagNumber(2)
  set phase(GamePhase v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasPhase() => $_has(1);
  @$pb.TagNumber(2)
  void clearPhase() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<Player> get players => $_getList(2);

  @$pb.TagNumber(4)
  $core.List<Card> get communityCards => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get pot => $_getI64(4);
  @$pb.TagNumber(5)
  set pot($fixnum.Int64 v) { $_setInt64(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasPot() => $_has(4);
  @$pb.TagNumber(5)
  void clearPot() => clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get currentBet => $_getI64(5);
  @$pb.TagNumber(6)
  set currentBet($fixnum.Int64 v) { $_setInt64(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasCurrentBet() => $_has(5);
  @$pb.TagNumber(6)
  void clearCurrentBet() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get currentPlayer => $_getSZ(6);
  @$pb.TagNumber(7)
  set currentPlayer($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasCurrentPlayer() => $_has(6);
  @$pb.TagNumber(7)
  void clearCurrentPlayer() => clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get minRaise => $_getI64(7);
  @$pb.TagNumber(8)
  set minRaise($fixnum.Int64 v) { $_setInt64(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasMinRaise() => $_has(7);
  @$pb.TagNumber(8)
  void clearMinRaise() => clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get maxRaise => $_getI64(8);
  @$pb.TagNumber(9)
  set maxRaise($fixnum.Int64 v) { $_setInt64(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasMaxRaise() => $_has(8);
  @$pb.TagNumber(9)
  void clearMaxRaise() => clearField(9);

  @$pb.TagNumber(10)
  $core.bool get gameStarted => $_getBF(9);
  @$pb.TagNumber(10)
  set gameStarted($core.bool v) { $_setBool(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasGameStarted() => $_has(9);
  @$pb.TagNumber(10)
  void clearGameStarted() => clearField(10);

  @$pb.TagNumber(11)
  $core.int get playersRequired => $_getIZ(10);
  @$pb.TagNumber(11)
  set playersRequired($core.int v) { $_setSignedInt32(10, v); }
  @$pb.TagNumber(11)
  $core.bool hasPlayersRequired() => $_has(10);
  @$pb.TagNumber(11)
  void clearPlayersRequired() => clearField(11);

  @$pb.TagNumber(12)
  $core.int get playersJoined => $_getIZ(11);
  @$pb.TagNumber(12)
  set playersJoined($core.int v) { $_setSignedInt32(11, v); }
  @$pb.TagNumber(12)
  $core.bool hasPlayersJoined() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlayersJoined() => clearField(12);

  @$pb.TagNumber(13)
  $core.String get phaseName => $_getSZ(12);
  @$pb.TagNumber(13)
  set phaseName($core.String v) { $_setString(12, v); }
  @$pb.TagNumber(13)
  $core.bool hasPhaseName() => $_has(12);
  @$pb.TagNumber(13)
  void clearPhaseName() => clearField(13);
}

class MakeBetRequest extends $pb.GeneratedMessage {
  factory MakeBetRequest({
    $core.String? playerId,
    $core.String? tableId,
    $fixnum.Int64? amount,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    if (amount != null) {
      $result.amount = amount;
    }
    return $result;
  }
  MakeBetRequest._() : super();
  factory MakeBetRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory MakeBetRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'MakeBetRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..aInt64(3, _omitFieldNames ? '' : 'amount')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  MakeBetRequest clone() => MakeBetRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  MakeBetRequest copyWith(void Function(MakeBetRequest) updates) => super.copyWith((message) => updates(message as MakeBetRequest)) as MakeBetRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MakeBetRequest create() => MakeBetRequest._();
  MakeBetRequest createEmptyInstance() => create();
  static $pb.PbList<MakeBetRequest> createRepeated() => $pb.PbList<MakeBetRequest>();
  @$core.pragma('dart2js:noInline')
  static MakeBetRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MakeBetRequest>(create);
  static MakeBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get amount => $_getI64(2);
  @$pb.TagNumber(3)
  set amount($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasAmount() => $_has(2);
  @$pb.TagNumber(3)
  void clearAmount() => clearField(3);
}

class MakeBetResponse extends $pb.GeneratedMessage {
  factory MakeBetResponse({
    $core.bool? success,
    $core.String? message,
    $fixnum.Int64? newBalance,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    if (newBalance != null) {
      $result.newBalance = newBalance;
    }
    return $result;
  }
  MakeBetResponse._() : super();
  factory MakeBetResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory MakeBetResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'MakeBetResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aInt64(3, _omitFieldNames ? '' : 'newBalance')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  MakeBetResponse clone() => MakeBetResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  MakeBetResponse copyWith(void Function(MakeBetResponse) updates) => super.copyWith((message) => updates(message as MakeBetResponse)) as MakeBetResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MakeBetResponse create() => MakeBetResponse._();
  MakeBetResponse createEmptyInstance() => create();
  static $pb.PbList<MakeBetResponse> createRepeated() => $pb.PbList<MakeBetResponse>();
  @$core.pragma('dart2js:noInline')
  static MakeBetResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MakeBetResponse>(create);
  static MakeBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get newBalance => $_getI64(2);
  @$pb.TagNumber(3)
  set newBalance($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasNewBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewBalance() => clearField(3);
}

class FoldBetRequest extends $pb.GeneratedMessage {
  factory FoldBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  FoldBetRequest._() : super();
  factory FoldBetRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FoldBetRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FoldBetRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FoldBetRequest clone() => FoldBetRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FoldBetRequest copyWith(void Function(FoldBetRequest) updates) => super.copyWith((message) => updates(message as FoldBetRequest)) as FoldBetRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FoldBetRequest create() => FoldBetRequest._();
  FoldBetRequest createEmptyInstance() => create();
  static $pb.PbList<FoldBetRequest> createRepeated() => $pb.PbList<FoldBetRequest>();
  @$core.pragma('dart2js:noInline')
  static FoldBetRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FoldBetRequest>(create);
  static FoldBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class FoldBetResponse extends $pb.GeneratedMessage {
  factory FoldBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  FoldBetResponse._() : super();
  factory FoldBetResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FoldBetResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FoldBetResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FoldBetResponse clone() => FoldBetResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FoldBetResponse copyWith(void Function(FoldBetResponse) updates) => super.copyWith((message) => updates(message as FoldBetResponse)) as FoldBetResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FoldBetResponse create() => FoldBetResponse._();
  FoldBetResponse createEmptyInstance() => create();
  static $pb.PbList<FoldBetResponse> createRepeated() => $pb.PbList<FoldBetResponse>();
  @$core.pragma('dart2js:noInline')
  static FoldBetResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FoldBetResponse>(create);
  static FoldBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class CheckBetRequest extends $pb.GeneratedMessage {
  factory CheckBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  CheckBetRequest._() : super();
  factory CheckBetRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CheckBetRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CheckBetRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CheckBetRequest clone() => CheckBetRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CheckBetRequest copyWith(void Function(CheckBetRequest) updates) => super.copyWith((message) => updates(message as CheckBetRequest)) as CheckBetRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CheckBetRequest create() => CheckBetRequest._();
  CheckBetRequest createEmptyInstance() => create();
  static $pb.PbList<CheckBetRequest> createRepeated() => $pb.PbList<CheckBetRequest>();
  @$core.pragma('dart2js:noInline')
  static CheckBetRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CheckBetRequest>(create);
  static CheckBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class CheckBetResponse extends $pb.GeneratedMessage {
  factory CheckBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  CheckBetResponse._() : super();
  factory CheckBetResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CheckBetResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CheckBetResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CheckBetResponse clone() => CheckBetResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CheckBetResponse copyWith(void Function(CheckBetResponse) updates) => super.copyWith((message) => updates(message as CheckBetResponse)) as CheckBetResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CheckBetResponse create() => CheckBetResponse._();
  CheckBetResponse createEmptyInstance() => create();
  static $pb.PbList<CheckBetResponse> createRepeated() => $pb.PbList<CheckBetResponse>();
  @$core.pragma('dart2js:noInline')
  static CheckBetResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CheckBetResponse>(create);
  static CheckBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class CallBetRequest extends $pb.GeneratedMessage {
  factory CallBetRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  CallBetRequest._() : super();
  factory CallBetRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CallBetRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CallBetRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CallBetRequest clone() => CallBetRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CallBetRequest copyWith(void Function(CallBetRequest) updates) => super.copyWith((message) => updates(message as CallBetRequest)) as CallBetRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CallBetRequest create() => CallBetRequest._();
  CallBetRequest createEmptyInstance() => create();
  static $pb.PbList<CallBetRequest> createRepeated() => $pb.PbList<CallBetRequest>();
  @$core.pragma('dart2js:noInline')
  static CallBetRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CallBetRequest>(create);
  static CallBetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class CallBetResponse extends $pb.GeneratedMessage {
  factory CallBetResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  CallBetResponse._() : super();
  factory CallBetResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CallBetResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CallBetResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CallBetResponse clone() => CallBetResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CallBetResponse copyWith(void Function(CallBetResponse) updates) => super.copyWith((message) => updates(message as CallBetResponse)) as CallBetResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CallBetResponse create() => CallBetResponse._();
  CallBetResponse createEmptyInstance() => create();
  static $pb.PbList<CallBetResponse> createRepeated() => $pb.PbList<CallBetResponse>();
  @$core.pragma('dart2js:noInline')
  static CallBetResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CallBetResponse>(create);
  static CallBetResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class GetGameStateRequest extends $pb.GeneratedMessage {
  factory GetGameStateRequest({
    $core.String? tableId,
  }) {
    final $result = create();
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  GetGameStateRequest._() : super();
  factory GetGameStateRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetGameStateRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetGameStateRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetGameStateRequest clone() => GetGameStateRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetGameStateRequest copyWith(void Function(GetGameStateRequest) updates) => super.copyWith((message) => updates(message as GetGameStateRequest)) as GetGameStateRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetGameStateRequest create() => GetGameStateRequest._();
  GetGameStateRequest createEmptyInstance() => create();
  static $pb.PbList<GetGameStateRequest> createRepeated() => $pb.PbList<GetGameStateRequest>();
  @$core.pragma('dart2js:noInline')
  static GetGameStateRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetGameStateRequest>(create);
  static GetGameStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => clearField(1);
}

class GetGameStateResponse extends $pb.GeneratedMessage {
  factory GetGameStateResponse({
    GameUpdate? gameState,
  }) {
    final $result = create();
    if (gameState != null) {
      $result.gameState = gameState;
    }
    return $result;
  }
  GetGameStateResponse._() : super();
  factory GetGameStateResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetGameStateResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetGameStateResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOM<GameUpdate>(1, _omitFieldNames ? '' : 'gameState', subBuilder: GameUpdate.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetGameStateResponse clone() => GetGameStateResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetGameStateResponse copyWith(void Function(GetGameStateResponse) updates) => super.copyWith((message) => updates(message as GetGameStateResponse)) as GetGameStateResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetGameStateResponse create() => GetGameStateResponse._();
  GetGameStateResponse createEmptyInstance() => create();
  static $pb.PbList<GetGameStateResponse> createRepeated() => $pb.PbList<GetGameStateResponse>();
  @$core.pragma('dart2js:noInline')
  static GetGameStateResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetGameStateResponse>(create);
  static GetGameStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  GameUpdate get gameState => $_getN(0);
  @$pb.TagNumber(1)
  set gameState(GameUpdate v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasGameState() => $_has(0);
  @$pb.TagNumber(1)
  void clearGameState() => clearField(1);
  @$pb.TagNumber(1)
  GameUpdate ensureGameState() => $_ensure(0);
}

class EvaluateHandRequest extends $pb.GeneratedMessage {
  factory EvaluateHandRequest({
    $core.Iterable<Card>? cards,
  }) {
    final $result = create();
    if (cards != null) {
      $result.cards.addAll(cards);
    }
    return $result;
  }
  EvaluateHandRequest._() : super();
  factory EvaluateHandRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory EvaluateHandRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'EvaluateHandRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..pc<Card>(1, _omitFieldNames ? '' : 'cards', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  EvaluateHandRequest clone() => EvaluateHandRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  EvaluateHandRequest copyWith(void Function(EvaluateHandRequest) updates) => super.copyWith((message) => updates(message as EvaluateHandRequest)) as EvaluateHandRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EvaluateHandRequest create() => EvaluateHandRequest._();
  EvaluateHandRequest createEmptyInstance() => create();
  static $pb.PbList<EvaluateHandRequest> createRepeated() => $pb.PbList<EvaluateHandRequest>();
  @$core.pragma('dart2js:noInline')
  static EvaluateHandRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EvaluateHandRequest>(create);
  static EvaluateHandRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<Card> get cards => $_getList(0);
}

class EvaluateHandResponse extends $pb.GeneratedMessage {
  factory EvaluateHandResponse({
    HandRank? rank,
    $core.String? description,
    $core.Iterable<Card>? bestHand,
  }) {
    final $result = create();
    if (rank != null) {
      $result.rank = rank;
    }
    if (description != null) {
      $result.description = description;
    }
    if (bestHand != null) {
      $result.bestHand.addAll(bestHand);
    }
    return $result;
  }
  EvaluateHandResponse._() : super();
  factory EvaluateHandResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory EvaluateHandResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'EvaluateHandResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..e<HandRank>(1, _omitFieldNames ? '' : 'rank', $pb.PbFieldType.OE, defaultOrMaker: HandRank.HIGH_CARD, valueOf: HandRank.valueOf, enumValues: HandRank.values)
    ..aOS(2, _omitFieldNames ? '' : 'description')
    ..pc<Card>(3, _omitFieldNames ? '' : 'bestHand', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  EvaluateHandResponse clone() => EvaluateHandResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  EvaluateHandResponse copyWith(void Function(EvaluateHandResponse) updates) => super.copyWith((message) => updates(message as EvaluateHandResponse)) as EvaluateHandResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EvaluateHandResponse create() => EvaluateHandResponse._();
  EvaluateHandResponse createEmptyInstance() => create();
  static $pb.PbList<EvaluateHandResponse> createRepeated() => $pb.PbList<EvaluateHandResponse>();
  @$core.pragma('dart2js:noInline')
  static EvaluateHandResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EvaluateHandResponse>(create);
  static EvaluateHandResponse? _defaultInstance;

  @$pb.TagNumber(1)
  HandRank get rank => $_getN(0);
  @$pb.TagNumber(1)
  set rank(HandRank v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasRank() => $_has(0);
  @$pb.TagNumber(1)
  void clearRank() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get description => $_getSZ(1);
  @$pb.TagNumber(2)
  set description($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasDescription() => $_has(1);
  @$pb.TagNumber(2)
  void clearDescription() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<Card> get bestHand => $_getList(2);
}

class GetLastWinnersRequest extends $pb.GeneratedMessage {
  factory GetLastWinnersRequest({
    $core.String? tableId,
  }) {
    final $result = create();
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  GetLastWinnersRequest._() : super();
  factory GetLastWinnersRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetLastWinnersRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetLastWinnersRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetLastWinnersRequest clone() => GetLastWinnersRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetLastWinnersRequest copyWith(void Function(GetLastWinnersRequest) updates) => super.copyWith((message) => updates(message as GetLastWinnersRequest)) as GetLastWinnersRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLastWinnersRequest create() => GetLastWinnersRequest._();
  GetLastWinnersRequest createEmptyInstance() => create();
  static $pb.PbList<GetLastWinnersRequest> createRepeated() => $pb.PbList<GetLastWinnersRequest>();
  @$core.pragma('dart2js:noInline')
  static GetLastWinnersRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetLastWinnersRequest>(create);
  static GetLastWinnersRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => clearField(1);
}

class GetLastWinnersResponse extends $pb.GeneratedMessage {
  factory GetLastWinnersResponse({
    $core.Iterable<Winner>? winners,
  }) {
    final $result = create();
    if (winners != null) {
      $result.winners.addAll(winners);
    }
    return $result;
  }
  GetLastWinnersResponse._() : super();
  factory GetLastWinnersResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetLastWinnersResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetLastWinnersResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..pc<Winner>(1, _omitFieldNames ? '' : 'winners', $pb.PbFieldType.PM, subBuilder: Winner.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetLastWinnersResponse clone() => GetLastWinnersResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetLastWinnersResponse copyWith(void Function(GetLastWinnersResponse) updates) => super.copyWith((message) => updates(message as GetLastWinnersResponse)) as GetLastWinnersResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLastWinnersResponse create() => GetLastWinnersResponse._();
  GetLastWinnersResponse createEmptyInstance() => create();
  static $pb.PbList<GetLastWinnersResponse> createRepeated() => $pb.PbList<GetLastWinnersResponse>();
  @$core.pragma('dart2js:noInline')
  static GetLastWinnersResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetLastWinnersResponse>(create);
  static GetLastWinnersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<Winner> get winners => $_getList(0);
}

class Winner extends $pb.GeneratedMessage {
  factory Winner({
    $core.String? playerId,
    HandRank? handRank,
    $core.Iterable<Card>? bestHand,
    $fixnum.Int64? winnings,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (handRank != null) {
      $result.handRank = handRank;
    }
    if (bestHand != null) {
      $result.bestHand.addAll(bestHand);
    }
    if (winnings != null) {
      $result.winnings = winnings;
    }
    return $result;
  }
  Winner._() : super();
  factory Winner.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Winner.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Winner', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..e<HandRank>(2, _omitFieldNames ? '' : 'handRank', $pb.PbFieldType.OE, defaultOrMaker: HandRank.HIGH_CARD, valueOf: HandRank.valueOf, enumValues: HandRank.values)
    ..pc<Card>(3, _omitFieldNames ? '' : 'bestHand', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..aInt64(4, _omitFieldNames ? '' : 'winnings')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Winner clone() => Winner()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Winner copyWith(void Function(Winner) updates) => super.copyWith((message) => updates(message as Winner)) as Winner;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Winner create() => Winner._();
  Winner createEmptyInstance() => create();
  static $pb.PbList<Winner> createRepeated() => $pb.PbList<Winner>();
  @$core.pragma('dart2js:noInline')
  static Winner getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Winner>(create);
  static Winner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  HandRank get handRank => $_getN(1);
  @$pb.TagNumber(2)
  set handRank(HandRank v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasHandRank() => $_has(1);
  @$pb.TagNumber(2)
  void clearHandRank() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<Card> get bestHand => $_getList(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get winnings => $_getI64(3);
  @$pb.TagNumber(4)
  set winnings($fixnum.Int64 v) { $_setInt64(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasWinnings() => $_has(3);
  @$pb.TagNumber(4)
  void clearWinnings() => clearField(4);
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
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (smallBlind != null) {
      $result.smallBlind = smallBlind;
    }
    if (bigBlind != null) {
      $result.bigBlind = bigBlind;
    }
    if (maxPlayers != null) {
      $result.maxPlayers = maxPlayers;
    }
    if (minPlayers != null) {
      $result.minPlayers = minPlayers;
    }
    if (minBalance != null) {
      $result.minBalance = minBalance;
    }
    if (buyIn != null) {
      $result.buyIn = buyIn;
    }
    if (startingChips != null) {
      $result.startingChips = startingChips;
    }
    if (timeBankSeconds != null) {
      $result.timeBankSeconds = timeBankSeconds;
    }
    if (autoStartMs != null) {
      $result.autoStartMs = autoStartMs;
    }
    return $result;
  }
  CreateTableRequest._() : super();
  factory CreateTableRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CreateTableRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CreateTableRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aInt64(2, _omitFieldNames ? '' : 'smallBlind')
    ..aInt64(3, _omitFieldNames ? '' : 'bigBlind')
    ..a<$core.int>(4, _omitFieldNames ? '' : 'maxPlayers', $pb.PbFieldType.O3)
    ..a<$core.int>(5, _omitFieldNames ? '' : 'minPlayers', $pb.PbFieldType.O3)
    ..aInt64(6, _omitFieldNames ? '' : 'minBalance')
    ..aInt64(7, _omitFieldNames ? '' : 'buyIn')
    ..aInt64(8, _omitFieldNames ? '' : 'startingChips')
    ..a<$core.int>(9, _omitFieldNames ? '' : 'timeBankSeconds', $pb.PbFieldType.O3)
    ..a<$core.int>(10, _omitFieldNames ? '' : 'autoStartMs', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CreateTableRequest clone() => CreateTableRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CreateTableRequest copyWith(void Function(CreateTableRequest) updates) => super.copyWith((message) => updates(message as CreateTableRequest)) as CreateTableRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateTableRequest create() => CreateTableRequest._();
  CreateTableRequest createEmptyInstance() => create();
  static $pb.PbList<CreateTableRequest> createRepeated() => $pb.PbList<CreateTableRequest>();
  @$core.pragma('dart2js:noInline')
  static CreateTableRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateTableRequest>(create);
  static CreateTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get smallBlind => $_getI64(1);
  @$pb.TagNumber(2)
  set smallBlind($fixnum.Int64 v) { $_setInt64(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasSmallBlind() => $_has(1);
  @$pb.TagNumber(2)
  void clearSmallBlind() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get bigBlind => $_getI64(2);
  @$pb.TagNumber(3)
  set bigBlind($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasBigBlind() => $_has(2);
  @$pb.TagNumber(3)
  void clearBigBlind() => clearField(3);

  @$pb.TagNumber(4)
  $core.int get maxPlayers => $_getIZ(3);
  @$pb.TagNumber(4)
  set maxPlayers($core.int v) { $_setSignedInt32(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasMaxPlayers() => $_has(3);
  @$pb.TagNumber(4)
  void clearMaxPlayers() => clearField(4);

  @$pb.TagNumber(5)
  $core.int get minPlayers => $_getIZ(4);
  @$pb.TagNumber(5)
  set minPlayers($core.int v) { $_setSignedInt32(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasMinPlayers() => $_has(4);
  @$pb.TagNumber(5)
  void clearMinPlayers() => clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get minBalance => $_getI64(5);
  @$pb.TagNumber(6)
  set minBalance($fixnum.Int64 v) { $_setInt64(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasMinBalance() => $_has(5);
  @$pb.TagNumber(6)
  void clearMinBalance() => clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get buyIn => $_getI64(6);
  @$pb.TagNumber(7)
  set buyIn($fixnum.Int64 v) { $_setInt64(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasBuyIn() => $_has(6);
  @$pb.TagNumber(7)
  void clearBuyIn() => clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get startingChips => $_getI64(7);
  @$pb.TagNumber(8)
  set startingChips($fixnum.Int64 v) { $_setInt64(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasStartingChips() => $_has(7);
  @$pb.TagNumber(8)
  void clearStartingChips() => clearField(8);

  @$pb.TagNumber(9)
  $core.int get timeBankSeconds => $_getIZ(8);
  @$pb.TagNumber(9)
  set timeBankSeconds($core.int v) { $_setSignedInt32(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasTimeBankSeconds() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimeBankSeconds() => clearField(9);

  @$pb.TagNumber(10)
  $core.int get autoStartMs => $_getIZ(9);
  @$pb.TagNumber(10)
  set autoStartMs($core.int v) { $_setSignedInt32(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasAutoStartMs() => $_has(9);
  @$pb.TagNumber(10)
  void clearAutoStartMs() => clearField(10);
}

class CreateTableResponse extends $pb.GeneratedMessage {
  factory CreateTableResponse({
    $core.String? tableId,
  }) {
    final $result = create();
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  CreateTableResponse._() : super();
  factory CreateTableResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CreateTableResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CreateTableResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CreateTableResponse clone() => CreateTableResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CreateTableResponse copyWith(void Function(CreateTableResponse) updates) => super.copyWith((message) => updates(message as CreateTableResponse)) as CreateTableResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateTableResponse create() => CreateTableResponse._();
  CreateTableResponse createEmptyInstance() => create();
  static $pb.PbList<CreateTableResponse> createRepeated() => $pb.PbList<CreateTableResponse>();
  @$core.pragma('dart2js:noInline')
  static CreateTableResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateTableResponse>(create);
  static CreateTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => clearField(1);
}

class JoinTableRequest extends $pb.GeneratedMessage {
  factory JoinTableRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  JoinTableRequest._() : super();
  factory JoinTableRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory JoinTableRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'JoinTableRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  JoinTableRequest clone() => JoinTableRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  JoinTableRequest copyWith(void Function(JoinTableRequest) updates) => super.copyWith((message) => updates(message as JoinTableRequest)) as JoinTableRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinTableRequest create() => JoinTableRequest._();
  JoinTableRequest createEmptyInstance() => create();
  static $pb.PbList<JoinTableRequest> createRepeated() => $pb.PbList<JoinTableRequest>();
  @$core.pragma('dart2js:noInline')
  static JoinTableRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<JoinTableRequest>(create);
  static JoinTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class JoinTableResponse extends $pb.GeneratedMessage {
  factory JoinTableResponse({
    $core.bool? success,
    $core.String? message,
    $fixnum.Int64? newBalance,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    if (newBalance != null) {
      $result.newBalance = newBalance;
    }
    return $result;
  }
  JoinTableResponse._() : super();
  factory JoinTableResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory JoinTableResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'JoinTableResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aInt64(3, _omitFieldNames ? '' : 'newBalance')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  JoinTableResponse clone() => JoinTableResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  JoinTableResponse copyWith(void Function(JoinTableResponse) updates) => super.copyWith((message) => updates(message as JoinTableResponse)) as JoinTableResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinTableResponse create() => JoinTableResponse._();
  JoinTableResponse createEmptyInstance() => create();
  static $pb.PbList<JoinTableResponse> createRepeated() => $pb.PbList<JoinTableResponse>();
  @$core.pragma('dart2js:noInline')
  static JoinTableResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<JoinTableResponse>(create);
  static JoinTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get newBalance => $_getI64(2);
  @$pb.TagNumber(3)
  set newBalance($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasNewBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewBalance() => clearField(3);
}

class LeaveTableRequest extends $pb.GeneratedMessage {
  factory LeaveTableRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  LeaveTableRequest._() : super();
  factory LeaveTableRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory LeaveTableRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'LeaveTableRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  LeaveTableRequest clone() => LeaveTableRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  LeaveTableRequest copyWith(void Function(LeaveTableRequest) updates) => super.copyWith((message) => updates(message as LeaveTableRequest)) as LeaveTableRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LeaveTableRequest create() => LeaveTableRequest._();
  LeaveTableRequest createEmptyInstance() => create();
  static $pb.PbList<LeaveTableRequest> createRepeated() => $pb.PbList<LeaveTableRequest>();
  @$core.pragma('dart2js:noInline')
  static LeaveTableRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<LeaveTableRequest>(create);
  static LeaveTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class LeaveTableResponse extends $pb.GeneratedMessage {
  factory LeaveTableResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  LeaveTableResponse._() : super();
  factory LeaveTableResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory LeaveTableResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'LeaveTableResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  LeaveTableResponse clone() => LeaveTableResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  LeaveTableResponse copyWith(void Function(LeaveTableResponse) updates) => super.copyWith((message) => updates(message as LeaveTableResponse)) as LeaveTableResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LeaveTableResponse create() => LeaveTableResponse._();
  LeaveTableResponse createEmptyInstance() => create();
  static $pb.PbList<LeaveTableResponse> createRepeated() => $pb.PbList<LeaveTableResponse>();
  @$core.pragma('dart2js:noInline')
  static LeaveTableResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<LeaveTableResponse>(create);
  static LeaveTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class GetTablesRequest extends $pb.GeneratedMessage {
  factory GetTablesRequest() => create();
  GetTablesRequest._() : super();
  factory GetTablesRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetTablesRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetTablesRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetTablesRequest clone() => GetTablesRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetTablesRequest copyWith(void Function(GetTablesRequest) updates) => super.copyWith((message) => updates(message as GetTablesRequest)) as GetTablesRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTablesRequest create() => GetTablesRequest._();
  GetTablesRequest createEmptyInstance() => create();
  static $pb.PbList<GetTablesRequest> createRepeated() => $pb.PbList<GetTablesRequest>();
  @$core.pragma('dart2js:noInline')
  static GetTablesRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetTablesRequest>(create);
  static GetTablesRequest? _defaultInstance;
}

class GetTablesResponse extends $pb.GeneratedMessage {
  factory GetTablesResponse({
    $core.Iterable<Table>? tables,
  }) {
    final $result = create();
    if (tables != null) {
      $result.tables.addAll(tables);
    }
    return $result;
  }
  GetTablesResponse._() : super();
  factory GetTablesResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetTablesResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetTablesResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..pc<Table>(1, _omitFieldNames ? '' : 'tables', $pb.PbFieldType.PM, subBuilder: Table.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetTablesResponse clone() => GetTablesResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetTablesResponse copyWith(void Function(GetTablesResponse) updates) => super.copyWith((message) => updates(message as GetTablesResponse)) as GetTablesResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTablesResponse create() => GetTablesResponse._();
  GetTablesResponse createEmptyInstance() => create();
  static $pb.PbList<GetTablesResponse> createRepeated() => $pb.PbList<GetTablesResponse>();
  @$core.pragma('dart2js:noInline')
  static GetTablesResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetTablesResponse>(create);
  static GetTablesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<Table> get tables => $_getList(0);
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
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (hostId != null) {
      $result.hostId = hostId;
    }
    if (players != null) {
      $result.players.addAll(players);
    }
    if (smallBlind != null) {
      $result.smallBlind = smallBlind;
    }
    if (bigBlind != null) {
      $result.bigBlind = bigBlind;
    }
    if (maxPlayers != null) {
      $result.maxPlayers = maxPlayers;
    }
    if (minPlayers != null) {
      $result.minPlayers = minPlayers;
    }
    if (currentPlayers != null) {
      $result.currentPlayers = currentPlayers;
    }
    if (minBalance != null) {
      $result.minBalance = minBalance;
    }
    if (buyIn != null) {
      $result.buyIn = buyIn;
    }
    if (phase != null) {
      $result.phase = phase;
    }
    if (gameStarted != null) {
      $result.gameStarted = gameStarted;
    }
    if (allPlayersReady != null) {
      $result.allPlayersReady = allPlayersReady;
    }
    return $result;
  }
  Table._() : super();
  factory Table.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Table.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Table', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'hostId')
    ..pc<Player>(3, _omitFieldNames ? '' : 'players', $pb.PbFieldType.PM, subBuilder: Player.create)
    ..aInt64(4, _omitFieldNames ? '' : 'smallBlind')
    ..aInt64(5, _omitFieldNames ? '' : 'bigBlind')
    ..a<$core.int>(6, _omitFieldNames ? '' : 'maxPlayers', $pb.PbFieldType.O3)
    ..a<$core.int>(7, _omitFieldNames ? '' : 'minPlayers', $pb.PbFieldType.O3)
    ..a<$core.int>(8, _omitFieldNames ? '' : 'currentPlayers', $pb.PbFieldType.O3)
    ..aInt64(9, _omitFieldNames ? '' : 'minBalance')
    ..aInt64(10, _omitFieldNames ? '' : 'buyIn')
    ..e<GamePhase>(11, _omitFieldNames ? '' : 'phase', $pb.PbFieldType.OE, defaultOrMaker: GamePhase.WAITING, valueOf: GamePhase.valueOf, enumValues: GamePhase.values)
    ..aOB(12, _omitFieldNames ? '' : 'gameStarted')
    ..aOB(13, _omitFieldNames ? '' : 'allPlayersReady')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Table clone() => Table()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Table copyWith(void Function(Table) updates) => super.copyWith((message) => updates(message as Table)) as Table;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Table create() => Table._();
  Table createEmptyInstance() => create();
  static $pb.PbList<Table> createRepeated() => $pb.PbList<Table>();
  @$core.pragma('dart2js:noInline')
  static Table getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Table>(create);
  static Table? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get hostId => $_getSZ(1);
  @$pb.TagNumber(2)
  set hostId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasHostId() => $_has(1);
  @$pb.TagNumber(2)
  void clearHostId() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<Player> get players => $_getList(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get smallBlind => $_getI64(3);
  @$pb.TagNumber(4)
  set smallBlind($fixnum.Int64 v) { $_setInt64(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasSmallBlind() => $_has(3);
  @$pb.TagNumber(4)
  void clearSmallBlind() => clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get bigBlind => $_getI64(4);
  @$pb.TagNumber(5)
  set bigBlind($fixnum.Int64 v) { $_setInt64(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasBigBlind() => $_has(4);
  @$pb.TagNumber(5)
  void clearBigBlind() => clearField(5);

  @$pb.TagNumber(6)
  $core.int get maxPlayers => $_getIZ(5);
  @$pb.TagNumber(6)
  set maxPlayers($core.int v) { $_setSignedInt32(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasMaxPlayers() => $_has(5);
  @$pb.TagNumber(6)
  void clearMaxPlayers() => clearField(6);

  @$pb.TagNumber(7)
  $core.int get minPlayers => $_getIZ(6);
  @$pb.TagNumber(7)
  set minPlayers($core.int v) { $_setSignedInt32(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasMinPlayers() => $_has(6);
  @$pb.TagNumber(7)
  void clearMinPlayers() => clearField(7);

  @$pb.TagNumber(8)
  $core.int get currentPlayers => $_getIZ(7);
  @$pb.TagNumber(8)
  set currentPlayers($core.int v) { $_setSignedInt32(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasCurrentPlayers() => $_has(7);
  @$pb.TagNumber(8)
  void clearCurrentPlayers() => clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get minBalance => $_getI64(8);
  @$pb.TagNumber(9)
  set minBalance($fixnum.Int64 v) { $_setInt64(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasMinBalance() => $_has(8);
  @$pb.TagNumber(9)
  void clearMinBalance() => clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get buyIn => $_getI64(9);
  @$pb.TagNumber(10)
  set buyIn($fixnum.Int64 v) { $_setInt64(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasBuyIn() => $_has(9);
  @$pb.TagNumber(10)
  void clearBuyIn() => clearField(10);

  @$pb.TagNumber(11)
  GamePhase get phase => $_getN(10);
  @$pb.TagNumber(11)
  set phase(GamePhase v) { setField(11, v); }
  @$pb.TagNumber(11)
  $core.bool hasPhase() => $_has(10);
  @$pb.TagNumber(11)
  void clearPhase() => clearField(11);

  @$pb.TagNumber(12)
  $core.bool get gameStarted => $_getBF(11);
  @$pb.TagNumber(12)
  set gameStarted($core.bool v) { $_setBool(11, v); }
  @$pb.TagNumber(12)
  $core.bool hasGameStarted() => $_has(11);
  @$pb.TagNumber(12)
  void clearGameStarted() => clearField(12);

  @$pb.TagNumber(13)
  $core.bool get allPlayersReady => $_getBF(12);
  @$pb.TagNumber(13)
  set allPlayersReady($core.bool v) { $_setBool(12, v); }
  @$pb.TagNumber(13)
  $core.bool hasAllPlayersReady() => $_has(12);
  @$pb.TagNumber(13)
  void clearAllPlayersReady() => clearField(13);
}

class GetBalanceRequest extends $pb.GeneratedMessage {
  factory GetBalanceRequest({
    $core.String? playerId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    return $result;
  }
  GetBalanceRequest._() : super();
  factory GetBalanceRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetBalanceRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetBalanceRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetBalanceRequest clone() => GetBalanceRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetBalanceRequest copyWith(void Function(GetBalanceRequest) updates) => super.copyWith((message) => updates(message as GetBalanceRequest)) as GetBalanceRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetBalanceRequest create() => GetBalanceRequest._();
  GetBalanceRequest createEmptyInstance() => create();
  static $pb.PbList<GetBalanceRequest> createRepeated() => $pb.PbList<GetBalanceRequest>();
  @$core.pragma('dart2js:noInline')
  static GetBalanceRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetBalanceRequest>(create);
  static GetBalanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);
}

class GetBalanceResponse extends $pb.GeneratedMessage {
  factory GetBalanceResponse({
    $fixnum.Int64? balance,
  }) {
    final $result = create();
    if (balance != null) {
      $result.balance = balance;
    }
    return $result;
  }
  GetBalanceResponse._() : super();
  factory GetBalanceResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetBalanceResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetBalanceResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'balance')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetBalanceResponse clone() => GetBalanceResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetBalanceResponse copyWith(void Function(GetBalanceResponse) updates) => super.copyWith((message) => updates(message as GetBalanceResponse)) as GetBalanceResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetBalanceResponse create() => GetBalanceResponse._();
  GetBalanceResponse createEmptyInstance() => create();
  static $pb.PbList<GetBalanceResponse> createRepeated() => $pb.PbList<GetBalanceResponse>();
  @$core.pragma('dart2js:noInline')
  static GetBalanceResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetBalanceResponse>(create);
  static GetBalanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get balance => $_getI64(0);
  @$pb.TagNumber(1)
  set balance($fixnum.Int64 v) { $_setInt64(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasBalance() => $_has(0);
  @$pb.TagNumber(1)
  void clearBalance() => clearField(1);
}

class UpdateBalanceRequest extends $pb.GeneratedMessage {
  factory UpdateBalanceRequest({
    $core.String? playerId,
    $fixnum.Int64? amount,
    $core.String? description,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (amount != null) {
      $result.amount = amount;
    }
    if (description != null) {
      $result.description = description;
    }
    return $result;
  }
  UpdateBalanceRequest._() : super();
  factory UpdateBalanceRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory UpdateBalanceRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'UpdateBalanceRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aInt64(2, _omitFieldNames ? '' : 'amount')
    ..aOS(3, _omitFieldNames ? '' : 'description')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  UpdateBalanceRequest clone() => UpdateBalanceRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  UpdateBalanceRequest copyWith(void Function(UpdateBalanceRequest) updates) => super.copyWith((message) => updates(message as UpdateBalanceRequest)) as UpdateBalanceRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateBalanceRequest create() => UpdateBalanceRequest._();
  UpdateBalanceRequest createEmptyInstance() => create();
  static $pb.PbList<UpdateBalanceRequest> createRepeated() => $pb.PbList<UpdateBalanceRequest>();
  @$core.pragma('dart2js:noInline')
  static UpdateBalanceRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UpdateBalanceRequest>(create);
  static UpdateBalanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get amount => $_getI64(1);
  @$pb.TagNumber(2)
  set amount($fixnum.Int64 v) { $_setInt64(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasAmount() => $_has(1);
  @$pb.TagNumber(2)
  void clearAmount() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get description => $_getSZ(2);
  @$pb.TagNumber(3)
  set description($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasDescription() => $_has(2);
  @$pb.TagNumber(3)
  void clearDescription() => clearField(3);
}

class UpdateBalanceResponse extends $pb.GeneratedMessage {
  factory UpdateBalanceResponse({
    $fixnum.Int64? newBalance,
    $core.String? message,
  }) {
    final $result = create();
    if (newBalance != null) {
      $result.newBalance = newBalance;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  UpdateBalanceResponse._() : super();
  factory UpdateBalanceResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory UpdateBalanceResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'UpdateBalanceResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'newBalance')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  UpdateBalanceResponse clone() => UpdateBalanceResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  UpdateBalanceResponse copyWith(void Function(UpdateBalanceResponse) updates) => super.copyWith((message) => updates(message as UpdateBalanceResponse)) as UpdateBalanceResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateBalanceResponse create() => UpdateBalanceResponse._();
  UpdateBalanceResponse createEmptyInstance() => create();
  static $pb.PbList<UpdateBalanceResponse> createRepeated() => $pb.PbList<UpdateBalanceResponse>();
  @$core.pragma('dart2js:noInline')
  static UpdateBalanceResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UpdateBalanceResponse>(create);
  static UpdateBalanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get newBalance => $_getI64(0);
  @$pb.TagNumber(1)
  set newBalance($fixnum.Int64 v) { $_setInt64(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasNewBalance() => $_has(0);
  @$pb.TagNumber(1)
  void clearNewBalance() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class ProcessTipRequest extends $pb.GeneratedMessage {
  factory ProcessTipRequest({
    $core.String? fromPlayerId,
    $core.String? toPlayerId,
    $fixnum.Int64? amount,
    $core.String? message,
  }) {
    final $result = create();
    if (fromPlayerId != null) {
      $result.fromPlayerId = fromPlayerId;
    }
    if (toPlayerId != null) {
      $result.toPlayerId = toPlayerId;
    }
    if (amount != null) {
      $result.amount = amount;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  ProcessTipRequest._() : super();
  factory ProcessTipRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ProcessTipRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ProcessTipRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'fromPlayerId')
    ..aOS(2, _omitFieldNames ? '' : 'toPlayerId')
    ..aInt64(3, _omitFieldNames ? '' : 'amount')
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ProcessTipRequest clone() => ProcessTipRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ProcessTipRequest copyWith(void Function(ProcessTipRequest) updates) => super.copyWith((message) => updates(message as ProcessTipRequest)) as ProcessTipRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProcessTipRequest create() => ProcessTipRequest._();
  ProcessTipRequest createEmptyInstance() => create();
  static $pb.PbList<ProcessTipRequest> createRepeated() => $pb.PbList<ProcessTipRequest>();
  @$core.pragma('dart2js:noInline')
  static ProcessTipRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ProcessTipRequest>(create);
  static ProcessTipRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get fromPlayerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set fromPlayerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasFromPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFromPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get toPlayerId => $_getSZ(1);
  @$pb.TagNumber(2)
  set toPlayerId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasToPlayerId() => $_has(1);
  @$pb.TagNumber(2)
  void clearToPlayerId() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get amount => $_getI64(2);
  @$pb.TagNumber(3)
  set amount($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasAmount() => $_has(2);
  @$pb.TagNumber(3)
  void clearAmount() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get message => $_getSZ(3);
  @$pb.TagNumber(4)
  set message($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessage() => clearField(4);
}

class ProcessTipResponse extends $pb.GeneratedMessage {
  factory ProcessTipResponse({
    $core.bool? success,
    $core.String? message,
    $fixnum.Int64? newBalance,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    if (newBalance != null) {
      $result.newBalance = newBalance;
    }
    return $result;
  }
  ProcessTipResponse._() : super();
  factory ProcessTipResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ProcessTipResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ProcessTipResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aInt64(3, _omitFieldNames ? '' : 'newBalance')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ProcessTipResponse clone() => ProcessTipResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ProcessTipResponse copyWith(void Function(ProcessTipResponse) updates) => super.copyWith((message) => updates(message as ProcessTipResponse)) as ProcessTipResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProcessTipResponse create() => ProcessTipResponse._();
  ProcessTipResponse createEmptyInstance() => create();
  static $pb.PbList<ProcessTipResponse> createRepeated() => $pb.PbList<ProcessTipResponse>();
  @$core.pragma('dart2js:noInline')
  static ProcessTipResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ProcessTipResponse>(create);
  static ProcessTipResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get newBalance => $_getI64(2);
  @$pb.TagNumber(3)
  set newBalance($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasNewBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewBalance() => clearField(3);
}

class StartNotificationStreamRequest extends $pb.GeneratedMessage {
  factory StartNotificationStreamRequest({
    $core.String? playerId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    return $result;
  }
  StartNotificationStreamRequest._() : super();
  factory StartNotificationStreamRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory StartNotificationStreamRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'StartNotificationStreamRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  StartNotificationStreamRequest clone() => StartNotificationStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  StartNotificationStreamRequest copyWith(void Function(StartNotificationStreamRequest) updates) => super.copyWith((message) => updates(message as StartNotificationStreamRequest)) as StartNotificationStreamRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartNotificationStreamRequest create() => StartNotificationStreamRequest._();
  StartNotificationStreamRequest createEmptyInstance() => create();
  static $pb.PbList<StartNotificationStreamRequest> createRepeated() => $pb.PbList<StartNotificationStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static StartNotificationStreamRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StartNotificationStreamRequest>(create);
  static StartNotificationStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);
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
  }) {
    final $result = create();
    if (type != null) {
      $result.type = type;
    }
    if (message != null) {
      $result.message = message;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (amount != null) {
      $result.amount = amount;
    }
    if (cards != null) {
      $result.cards.addAll(cards);
    }
    if (handRank != null) {
      $result.handRank = handRank;
    }
    if (newBalance != null) {
      $result.newBalance = newBalance;
    }
    if (table != null) {
      $result.table = table;
    }
    if (ready != null) {
      $result.ready = ready;
    }
    if (started != null) {
      $result.started = started;
    }
    if (gameReadyToPlay != null) {
      $result.gameReadyToPlay = gameReadyToPlay;
    }
    if (countdown != null) {
      $result.countdown = countdown;
    }
    if (winners != null) {
      $result.winners.addAll(winners);
    }
    if (showdown != null) {
      $result.showdown = showdown;
    }
    return $result;
  }
  Notification._() : super();
  factory Notification.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Notification.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Notification', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..e<NotificationType>(1, _omitFieldNames ? '' : 'type', $pb.PbFieldType.OE, defaultOrMaker: NotificationType.UNKNOWN, valueOf: NotificationType.valueOf, enumValues: NotificationType.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOS(3, _omitFieldNames ? '' : 'tableId')
    ..aOS(4, _omitFieldNames ? '' : 'playerId')
    ..aInt64(5, _omitFieldNames ? '' : 'amount')
    ..pc<Card>(6, _omitFieldNames ? '' : 'cards', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..e<HandRank>(7, _omitFieldNames ? '' : 'handRank', $pb.PbFieldType.OE, defaultOrMaker: HandRank.HIGH_CARD, valueOf: HandRank.valueOf, enumValues: HandRank.values)
    ..aInt64(8, _omitFieldNames ? '' : 'newBalance')
    ..aOM<Table>(9, _omitFieldNames ? '' : 'table', subBuilder: Table.create)
    ..aOB(10, _omitFieldNames ? '' : 'ready')
    ..aOB(11, _omitFieldNames ? '' : 'started')
    ..aOB(12, _omitFieldNames ? '' : 'gameReadyToPlay')
    ..a<$core.int>(13, _omitFieldNames ? '' : 'countdown', $pb.PbFieldType.O3)
    ..pc<Winner>(14, _omitFieldNames ? '' : 'winners', $pb.PbFieldType.PM, subBuilder: Winner.create)
    ..aOM<Showdown>(15, _omitFieldNames ? '' : 'showdown', subBuilder: Showdown.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Notification clone() => Notification()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Notification copyWith(void Function(Notification) updates) => super.copyWith((message) => updates(message as Notification)) as Notification;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Notification create() => Notification._();
  Notification createEmptyInstance() => create();
  static $pb.PbList<Notification> createRepeated() => $pb.PbList<Notification>();
  @$core.pragma('dart2js:noInline')
  static Notification getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Notification>(create);
  static Notification? _defaultInstance;

  @$pb.TagNumber(1)
  NotificationType get type => $_getN(0);
  @$pb.TagNumber(1)
  set type(NotificationType v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get tableId => $_getSZ(2);
  @$pb.TagNumber(3)
  set tableId($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasTableId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTableId() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get playerId => $_getSZ(3);
  @$pb.TagNumber(4)
  set playerId($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasPlayerId() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlayerId() => clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get amount => $_getI64(4);
  @$pb.TagNumber(5)
  set amount($fixnum.Int64 v) { $_setInt64(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasAmount() => $_has(4);
  @$pb.TagNumber(5)
  void clearAmount() => clearField(5);

  @$pb.TagNumber(6)
  $core.List<Card> get cards => $_getList(5);

  @$pb.TagNumber(7)
  HandRank get handRank => $_getN(6);
  @$pb.TagNumber(7)
  set handRank(HandRank v) { setField(7, v); }
  @$pb.TagNumber(7)
  $core.bool hasHandRank() => $_has(6);
  @$pb.TagNumber(7)
  void clearHandRank() => clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get newBalance => $_getI64(7);
  @$pb.TagNumber(8)
  set newBalance($fixnum.Int64 v) { $_setInt64(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasNewBalance() => $_has(7);
  @$pb.TagNumber(8)
  void clearNewBalance() => clearField(8);

  @$pb.TagNumber(9)
  Table get table => $_getN(8);
  @$pb.TagNumber(9)
  set table(Table v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasTable() => $_has(8);
  @$pb.TagNumber(9)
  void clearTable() => clearField(9);
  @$pb.TagNumber(9)
  Table ensureTable() => $_ensure(8);

  @$pb.TagNumber(10)
  $core.bool get ready => $_getBF(9);
  @$pb.TagNumber(10)
  set ready($core.bool v) { $_setBool(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasReady() => $_has(9);
  @$pb.TagNumber(10)
  void clearReady() => clearField(10);

  @$pb.TagNumber(11)
  $core.bool get started => $_getBF(10);
  @$pb.TagNumber(11)
  set started($core.bool v) { $_setBool(10, v); }
  @$pb.TagNumber(11)
  $core.bool hasStarted() => $_has(10);
  @$pb.TagNumber(11)
  void clearStarted() => clearField(11);

  @$pb.TagNumber(12)
  $core.bool get gameReadyToPlay => $_getBF(11);
  @$pb.TagNumber(12)
  set gameReadyToPlay($core.bool v) { $_setBool(11, v); }
  @$pb.TagNumber(12)
  $core.bool hasGameReadyToPlay() => $_has(11);
  @$pb.TagNumber(12)
  void clearGameReadyToPlay() => clearField(12);

  @$pb.TagNumber(13)
  $core.int get countdown => $_getIZ(12);
  @$pb.TagNumber(13)
  set countdown($core.int v) { $_setSignedInt32(12, v); }
  @$pb.TagNumber(13)
  $core.bool hasCountdown() => $_has(12);
  @$pb.TagNumber(13)
  void clearCountdown() => clearField(13);

  @$pb.TagNumber(14)
  $core.List<Winner> get winners => $_getList(13);

  @$pb.TagNumber(15)
  Showdown get showdown => $_getN(14);
  @$pb.TagNumber(15)
  set showdown(Showdown v) { setField(15, v); }
  @$pb.TagNumber(15)
  $core.bool hasShowdown() => $_has(14);
  @$pb.TagNumber(15)
  void clearShowdown() => clearField(15);
  @$pb.TagNumber(15)
  Showdown ensureShowdown() => $_ensure(14);
}

class Showdown extends $pb.GeneratedMessage {
  factory Showdown({
    $core.Iterable<Winner>? winners,
    $fixnum.Int64? pot,
  }) {
    final $result = create();
    if (winners != null) {
      $result.winners.addAll(winners);
    }
    if (pot != null) {
      $result.pot = pot;
    }
    return $result;
  }
  Showdown._() : super();
  factory Showdown.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Showdown.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Showdown', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..pc<Winner>(1, _omitFieldNames ? '' : 'winners', $pb.PbFieldType.PM, subBuilder: Winner.create)
    ..aInt64(2, _omitFieldNames ? '' : 'pot')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Showdown clone() => Showdown()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Showdown copyWith(void Function(Showdown) updates) => super.copyWith((message) => updates(message as Showdown)) as Showdown;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Showdown create() => Showdown._();
  Showdown createEmptyInstance() => create();
  static $pb.PbList<Showdown> createRepeated() => $pb.PbList<Showdown>();
  @$core.pragma('dart2js:noInline')
  static Showdown getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Showdown>(create);
  static Showdown? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<Winner> get winners => $_getList(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get pot => $_getI64(1);
  @$pb.TagNumber(2)
  set pot($fixnum.Int64 v) { $_setInt64(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasPot() => $_has(1);
  @$pb.TagNumber(2)
  void clearPot() => clearField(2);
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
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (name != null) {
      $result.name = name;
    }
    if (balance != null) {
      $result.balance = balance;
    }
    if (hand != null) {
      $result.hand.addAll(hand);
    }
    if (currentBet != null) {
      $result.currentBet = currentBet;
    }
    if (folded != null) {
      $result.folded = folded;
    }
    if (isTurn != null) {
      $result.isTurn = isTurn;
    }
    if (isAllIn != null) {
      $result.isAllIn = isAllIn;
    }
    if (isDealer != null) {
      $result.isDealer = isDealer;
    }
    if (isReady != null) {
      $result.isReady = isReady;
    }
    if (handDescription != null) {
      $result.handDescription = handDescription;
    }
    if (playerState != null) {
      $result.playerState = playerState;
    }
    return $result;
  }
  Player._() : super();
  factory Player.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Player.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Player', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aInt64(3, _omitFieldNames ? '' : 'balance')
    ..pc<Card>(4, _omitFieldNames ? '' : 'hand', $pb.PbFieldType.PM, subBuilder: Card.create)
    ..aInt64(5, _omitFieldNames ? '' : 'currentBet')
    ..aOB(6, _omitFieldNames ? '' : 'folded')
    ..aOB(7, _omitFieldNames ? '' : 'isTurn')
    ..aOB(8, _omitFieldNames ? '' : 'isAllIn')
    ..aOB(9, _omitFieldNames ? '' : 'isDealer')
    ..aOB(10, _omitFieldNames ? '' : 'isReady')
    ..aOS(11, _omitFieldNames ? '' : 'handDescription')
    ..e<PlayerState>(12, _omitFieldNames ? '' : 'playerState', $pb.PbFieldType.OE, defaultOrMaker: PlayerState.PLAYER_STATE_UNINITIALIZED, valueOf: PlayerState.valueOf, enumValues: PlayerState.values)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Player clone() => Player()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Player copyWith(void Function(Player) updates) => super.copyWith((message) => updates(message as Player)) as Player;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Player create() => Player._();
  Player createEmptyInstance() => create();
  static $pb.PbList<Player> createRepeated() => $pb.PbList<Player>();
  @$core.pragma('dart2js:noInline')
  static Player getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Player>(create);
  static Player? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get balance => $_getI64(2);
  @$pb.TagNumber(3)
  set balance($fixnum.Int64 v) { $_setInt64(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasBalance() => $_has(2);
  @$pb.TagNumber(3)
  void clearBalance() => clearField(3);

  @$pb.TagNumber(4)
  $core.List<Card> get hand => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get currentBet => $_getI64(4);
  @$pb.TagNumber(5)
  set currentBet($fixnum.Int64 v) { $_setInt64(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasCurrentBet() => $_has(4);
  @$pb.TagNumber(5)
  void clearCurrentBet() => clearField(5);

  @$pb.TagNumber(6)
  $core.bool get folded => $_getBF(5);
  @$pb.TagNumber(6)
  set folded($core.bool v) { $_setBool(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasFolded() => $_has(5);
  @$pb.TagNumber(6)
  void clearFolded() => clearField(6);

  @$pb.TagNumber(7)
  $core.bool get isTurn => $_getBF(6);
  @$pb.TagNumber(7)
  set isTurn($core.bool v) { $_setBool(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasIsTurn() => $_has(6);
  @$pb.TagNumber(7)
  void clearIsTurn() => clearField(7);

  @$pb.TagNumber(8)
  $core.bool get isAllIn => $_getBF(7);
  @$pb.TagNumber(8)
  set isAllIn($core.bool v) { $_setBool(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasIsAllIn() => $_has(7);
  @$pb.TagNumber(8)
  void clearIsAllIn() => clearField(8);

  @$pb.TagNumber(9)
  $core.bool get isDealer => $_getBF(8);
  @$pb.TagNumber(9)
  set isDealer($core.bool v) { $_setBool(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasIsDealer() => $_has(8);
  @$pb.TagNumber(9)
  void clearIsDealer() => clearField(9);

  @$pb.TagNumber(10)
  $core.bool get isReady => $_getBF(9);
  @$pb.TagNumber(10)
  set isReady($core.bool v) { $_setBool(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasIsReady() => $_has(9);
  @$pb.TagNumber(10)
  void clearIsReady() => clearField(10);

  @$pb.TagNumber(11)
  $core.String get handDescription => $_getSZ(10);
  @$pb.TagNumber(11)
  set handDescription($core.String v) { $_setString(10, v); }
  @$pb.TagNumber(11)
  $core.bool hasHandDescription() => $_has(10);
  @$pb.TagNumber(11)
  void clearHandDescription() => clearField(11);

  @$pb.TagNumber(12)
  PlayerState get playerState => $_getN(11);
  @$pb.TagNumber(12)
  set playerState(PlayerState v) { setField(12, v); }
  @$pb.TagNumber(12)
  $core.bool hasPlayerState() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlayerState() => clearField(12);
}

class Card extends $pb.GeneratedMessage {
  factory Card({
    $core.String? suit,
    $core.String? value,
  }) {
    final $result = create();
    if (suit != null) {
      $result.suit = suit;
    }
    if (value != null) {
      $result.value = value;
    }
    return $result;
  }
  Card._() : super();
  factory Card.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory Card.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Card', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'suit')
    ..aOS(2, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  Card clone() => Card()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  Card copyWith(void Function(Card) updates) => super.copyWith((message) => updates(message as Card)) as Card;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Card create() => Card._();
  Card createEmptyInstance() => create();
  static $pb.PbList<Card> createRepeated() => $pb.PbList<Card>();
  @$core.pragma('dart2js:noInline')
  static Card getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Card>(create);
  static Card? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get suit => $_getSZ(0);
  @$pb.TagNumber(1)
  set suit($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuit() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuit() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get value => $_getSZ(1);
  @$pb.TagNumber(2)
  set value($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(2)
  void clearValue() => clearField(2);
}

class SetPlayerReadyRequest extends $pb.GeneratedMessage {
  factory SetPlayerReadyRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  SetPlayerReadyRequest._() : super();
  factory SetPlayerReadyRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SetPlayerReadyRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SetPlayerReadyRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SetPlayerReadyRequest clone() => SetPlayerReadyRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SetPlayerReadyRequest copyWith(void Function(SetPlayerReadyRequest) updates) => super.copyWith((message) => updates(message as SetPlayerReadyRequest)) as SetPlayerReadyRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyRequest create() => SetPlayerReadyRequest._();
  SetPlayerReadyRequest createEmptyInstance() => create();
  static $pb.PbList<SetPlayerReadyRequest> createRepeated() => $pb.PbList<SetPlayerReadyRequest>();
  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SetPlayerReadyRequest>(create);
  static SetPlayerReadyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class SetPlayerReadyResponse extends $pb.GeneratedMessage {
  factory SetPlayerReadyResponse({
    $core.bool? success,
    $core.String? message,
    $core.bool? allPlayersReady,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    if (allPlayersReady != null) {
      $result.allPlayersReady = allPlayersReady;
    }
    return $result;
  }
  SetPlayerReadyResponse._() : super();
  factory SetPlayerReadyResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SetPlayerReadyResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SetPlayerReadyResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOB(3, _omitFieldNames ? '' : 'allPlayersReady')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SetPlayerReadyResponse clone() => SetPlayerReadyResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SetPlayerReadyResponse copyWith(void Function(SetPlayerReadyResponse) updates) => super.copyWith((message) => updates(message as SetPlayerReadyResponse)) as SetPlayerReadyResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyResponse create() => SetPlayerReadyResponse._();
  SetPlayerReadyResponse createEmptyInstance() => create();
  static $pb.PbList<SetPlayerReadyResponse> createRepeated() => $pb.PbList<SetPlayerReadyResponse>();
  @$core.pragma('dart2js:noInline')
  static SetPlayerReadyResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SetPlayerReadyResponse>(create);
  static SetPlayerReadyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);

  @$pb.TagNumber(3)
  $core.bool get allPlayersReady => $_getBF(2);
  @$pb.TagNumber(3)
  set allPlayersReady($core.bool v) { $_setBool(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasAllPlayersReady() => $_has(2);
  @$pb.TagNumber(3)
  void clearAllPlayersReady() => clearField(3);
}

class SetPlayerUnreadyRequest extends $pb.GeneratedMessage {
  factory SetPlayerUnreadyRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  SetPlayerUnreadyRequest._() : super();
  factory SetPlayerUnreadyRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SetPlayerUnreadyRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SetPlayerUnreadyRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SetPlayerUnreadyRequest clone() => SetPlayerUnreadyRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SetPlayerUnreadyRequest copyWith(void Function(SetPlayerUnreadyRequest) updates) => super.copyWith((message) => updates(message as SetPlayerUnreadyRequest)) as SetPlayerUnreadyRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyRequest create() => SetPlayerUnreadyRequest._();
  SetPlayerUnreadyRequest createEmptyInstance() => create();
  static $pb.PbList<SetPlayerUnreadyRequest> createRepeated() => $pb.PbList<SetPlayerUnreadyRequest>();
  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SetPlayerUnreadyRequest>(create);
  static SetPlayerUnreadyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class SetPlayerUnreadyResponse extends $pb.GeneratedMessage {
  factory SetPlayerUnreadyResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  SetPlayerUnreadyResponse._() : super();
  factory SetPlayerUnreadyResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SetPlayerUnreadyResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SetPlayerUnreadyResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SetPlayerUnreadyResponse clone() => SetPlayerUnreadyResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SetPlayerUnreadyResponse copyWith(void Function(SetPlayerUnreadyResponse) updates) => super.copyWith((message) => updates(message as SetPlayerUnreadyResponse)) as SetPlayerUnreadyResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyResponse create() => SetPlayerUnreadyResponse._();
  SetPlayerUnreadyResponse createEmptyInstance() => create();
  static $pb.PbList<SetPlayerUnreadyResponse> createRepeated() => $pb.PbList<SetPlayerUnreadyResponse>();
  @$core.pragma('dart2js:noInline')
  static SetPlayerUnreadyResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SetPlayerUnreadyResponse>(create);
  static SetPlayerUnreadyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class GetPlayerCurrentTableRequest extends $pb.GeneratedMessage {
  factory GetPlayerCurrentTableRequest({
    $core.String? playerId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    return $result;
  }
  GetPlayerCurrentTableRequest._() : super();
  factory GetPlayerCurrentTableRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetPlayerCurrentTableRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetPlayerCurrentTableRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetPlayerCurrentTableRequest clone() => GetPlayerCurrentTableRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetPlayerCurrentTableRequest copyWith(void Function(GetPlayerCurrentTableRequest) updates) => super.copyWith((message) => updates(message as GetPlayerCurrentTableRequest)) as GetPlayerCurrentTableRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableRequest create() => GetPlayerCurrentTableRequest._();
  GetPlayerCurrentTableRequest createEmptyInstance() => create();
  static $pb.PbList<GetPlayerCurrentTableRequest> createRepeated() => $pb.PbList<GetPlayerCurrentTableRequest>();
  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetPlayerCurrentTableRequest>(create);
  static GetPlayerCurrentTableRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);
}

class GetPlayerCurrentTableResponse extends $pb.GeneratedMessage {
  factory GetPlayerCurrentTableResponse({
    $core.String? tableId,
  }) {
    final $result = create();
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  GetPlayerCurrentTableResponse._() : super();
  factory GetPlayerCurrentTableResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetPlayerCurrentTableResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetPlayerCurrentTableResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetPlayerCurrentTableResponse clone() => GetPlayerCurrentTableResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetPlayerCurrentTableResponse copyWith(void Function(GetPlayerCurrentTableResponse) updates) => super.copyWith((message) => updates(message as GetPlayerCurrentTableResponse)) as GetPlayerCurrentTableResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableResponse create() => GetPlayerCurrentTableResponse._();
  GetPlayerCurrentTableResponse createEmptyInstance() => create();
  static $pb.PbList<GetPlayerCurrentTableResponse> createRepeated() => $pb.PbList<GetPlayerCurrentTableResponse>();
  @$core.pragma('dart2js:noInline')
  static GetPlayerCurrentTableResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetPlayerCurrentTableResponse>(create);
  static GetPlayerCurrentTableResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tableId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tableId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasTableId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTableId() => clearField(1);
}

class ShowCardsRequest extends $pb.GeneratedMessage {
  factory ShowCardsRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  ShowCardsRequest._() : super();
  factory ShowCardsRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ShowCardsRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ShowCardsRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ShowCardsRequest clone() => ShowCardsRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ShowCardsRequest copyWith(void Function(ShowCardsRequest) updates) => super.copyWith((message) => updates(message as ShowCardsRequest)) as ShowCardsRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShowCardsRequest create() => ShowCardsRequest._();
  ShowCardsRequest createEmptyInstance() => create();
  static $pb.PbList<ShowCardsRequest> createRepeated() => $pb.PbList<ShowCardsRequest>();
  @$core.pragma('dart2js:noInline')
  static ShowCardsRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ShowCardsRequest>(create);
  static ShowCardsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class ShowCardsResponse extends $pb.GeneratedMessage {
  factory ShowCardsResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  ShowCardsResponse._() : super();
  factory ShowCardsResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ShowCardsResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ShowCardsResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ShowCardsResponse clone() => ShowCardsResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ShowCardsResponse copyWith(void Function(ShowCardsResponse) updates) => super.copyWith((message) => updates(message as ShowCardsResponse)) as ShowCardsResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShowCardsResponse create() => ShowCardsResponse._();
  ShowCardsResponse createEmptyInstance() => create();
  static $pb.PbList<ShowCardsResponse> createRepeated() => $pb.PbList<ShowCardsResponse>();
  @$core.pragma('dart2js:noInline')
  static ShowCardsResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ShowCardsResponse>(create);
  static ShowCardsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}

class HideCardsRequest extends $pb.GeneratedMessage {
  factory HideCardsRequest({
    $core.String? playerId,
    $core.String? tableId,
  }) {
    final $result = create();
    if (playerId != null) {
      $result.playerId = playerId;
    }
    if (tableId != null) {
      $result.tableId = tableId;
    }
    return $result;
  }
  HideCardsRequest._() : super();
  factory HideCardsRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory HideCardsRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'HideCardsRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'playerId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  HideCardsRequest clone() => HideCardsRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  HideCardsRequest copyWith(void Function(HideCardsRequest) updates) => super.copyWith((message) => updates(message as HideCardsRequest)) as HideCardsRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HideCardsRequest create() => HideCardsRequest._();
  HideCardsRequest createEmptyInstance() => create();
  static $pb.PbList<HideCardsRequest> createRepeated() => $pb.PbList<HideCardsRequest>();
  @$core.pragma('dart2js:noInline')
  static HideCardsRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<HideCardsRequest>(create);
  static HideCardsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get playerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set playerId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasPlayerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlayerId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => clearField(2);
}

class HideCardsResponse extends $pb.GeneratedMessage {
  factory HideCardsResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  HideCardsResponse._() : super();
  factory HideCardsResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory HideCardsResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'HideCardsResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  HideCardsResponse clone() => HideCardsResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  HideCardsResponse copyWith(void Function(HideCardsResponse) updates) => super.copyWith((message) => updates(message as HideCardsResponse)) as HideCardsResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HideCardsResponse create() => HideCardsResponse._();
  HideCardsResponse createEmptyInstance() => create();
  static $pb.PbList<HideCardsResponse> createRepeated() => $pb.PbList<HideCardsResponse>();
  @$core.pragma('dart2js:noInline')
  static HideCardsResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<HideCardsResponse>(create);
  static HideCardsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
