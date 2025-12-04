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

import 'package:protobuf/protobuf.dart' as $pb;

/// Enums
class GamePhase extends $pb.ProtobufEnum {
  static const GamePhase WAITING =
      GamePhase._(0, _omitEnumNames ? '' : 'WAITING');
  static const GamePhase NEW_HAND_DEALING =
      GamePhase._(1, _omitEnumNames ? '' : 'NEW_HAND_DEALING');
  static const GamePhase PRE_FLOP =
      GamePhase._(2, _omitEnumNames ? '' : 'PRE_FLOP');
  static const GamePhase FLOP = GamePhase._(3, _omitEnumNames ? '' : 'FLOP');
  static const GamePhase TURN = GamePhase._(4, _omitEnumNames ? '' : 'TURN');
  static const GamePhase RIVER = GamePhase._(5, _omitEnumNames ? '' : 'RIVER');
  static const GamePhase SHOWDOWN =
      GamePhase._(6, _omitEnumNames ? '' : 'SHOWDOWN');

  static const $core.List<GamePhase> values = <GamePhase>[
    WAITING,
    NEW_HAND_DEALING,
    PRE_FLOP,
    FLOP,
    TURN,
    RIVER,
    SHOWDOWN,
  ];

  static final $core.List<GamePhase?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static GamePhase? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const GamePhase._(super.value, super.name);
}

/// PlayerState captures the per-player state machine state.
class PlayerState extends $pb.ProtobufEnum {
  static const PlayerState PLAYER_STATE_UNINITIALIZED =
      PlayerState._(0, _omitEnumNames ? '' : 'PLAYER_STATE_UNINITIALIZED');
  static const PlayerState PLAYER_STATE_AT_TABLE =
      PlayerState._(1, _omitEnumNames ? '' : 'PLAYER_STATE_AT_TABLE');
  static const PlayerState PLAYER_STATE_IN_GAME =
      PlayerState._(2, _omitEnumNames ? '' : 'PLAYER_STATE_IN_GAME');
  static const PlayerState PLAYER_STATE_ALL_IN =
      PlayerState._(3, _omitEnumNames ? '' : 'PLAYER_STATE_ALL_IN');
  static const PlayerState PLAYER_STATE_FOLDED =
      PlayerState._(4, _omitEnumNames ? '' : 'PLAYER_STATE_FOLDED');
  static const PlayerState PLAYER_STATE_LEFT =
      PlayerState._(5, _omitEnumNames ? '' : 'PLAYER_STATE_LEFT');

  static const $core.List<PlayerState> values = <PlayerState>[
    PLAYER_STATE_UNINITIALIZED,
    PLAYER_STATE_AT_TABLE,
    PLAYER_STATE_IN_GAME,
    PLAYER_STATE_ALL_IN,
    PLAYER_STATE_FOLDED,
    PLAYER_STATE_LEFT,
  ];

  static final $core.List<PlayerState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static PlayerState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PlayerState._(super.value, super.name);
}

class NotificationType extends $pb.ProtobufEnum {
  static const NotificationType UNKNOWN =
      NotificationType._(0, _omitEnumNames ? '' : 'UNKNOWN');
  static const NotificationType PLAYER_JOINED =
      NotificationType._(1, _omitEnumNames ? '' : 'PLAYER_JOINED');
  static const NotificationType PLAYER_LEFT =
      NotificationType._(2, _omitEnumNames ? '' : 'PLAYER_LEFT');
  static const NotificationType GAME_STARTED =
      NotificationType._(3, _omitEnumNames ? '' : 'GAME_STARTED');
  static const NotificationType GAME_ENDED =
      NotificationType._(4, _omitEnumNames ? '' : 'GAME_ENDED');
  static const NotificationType BET_MADE =
      NotificationType._(5, _omitEnumNames ? '' : 'BET_MADE');
  static const NotificationType PLAYER_FOLDED =
      NotificationType._(6, _omitEnumNames ? '' : 'PLAYER_FOLDED');
  static const NotificationType NEW_ROUND =
      NotificationType._(7, _omitEnumNames ? '' : 'NEW_ROUND');
  static const NotificationType SHOWDOWN_RESULT =
      NotificationType._(8, _omitEnumNames ? '' : 'SHOWDOWN_RESULT');
  static const NotificationType TIP_RECEIVED =
      NotificationType._(9, _omitEnumNames ? '' : 'TIP_RECEIVED');
  static const NotificationType BALANCE_UPDATED =
      NotificationType._(10, _omitEnumNames ? '' : 'BALANCE_UPDATED');
  static const NotificationType TABLE_CREATED =
      NotificationType._(11, _omitEnumNames ? '' : 'TABLE_CREATED');
  static const NotificationType TABLE_REMOVED =
      NotificationType._(12, _omitEnumNames ? '' : 'TABLE_REMOVED');
  static const NotificationType PLAYER_READY =
      NotificationType._(13, _omitEnumNames ? '' : 'PLAYER_READY');
  static const NotificationType PLAYER_UNREADY =
      NotificationType._(14, _omitEnumNames ? '' : 'PLAYER_UNREADY');
  static const NotificationType ALL_PLAYERS_READY =
      NotificationType._(15, _omitEnumNames ? '' : 'ALL_PLAYERS_READY');
  static const NotificationType SMALL_BLIND_POSTED =
      NotificationType._(16, _omitEnumNames ? '' : 'SMALL_BLIND_POSTED');
  static const NotificationType BIG_BLIND_POSTED =
      NotificationType._(17, _omitEnumNames ? '' : 'BIG_BLIND_POSTED');
  static const NotificationType CALL_MADE =
      NotificationType._(18, _omitEnumNames ? '' : 'CALL_MADE');
  static const NotificationType CHECK_MADE =
      NotificationType._(19, _omitEnumNames ? '' : 'CHECK_MADE');
  static const NotificationType CARDS_SHOWN =
      NotificationType._(20, _omitEnumNames ? '' : 'CARDS_SHOWN');
  static const NotificationType CARDS_HIDDEN =
      NotificationType._(21, _omitEnumNames ? '' : 'CARDS_HIDDEN');
  static const NotificationType NEW_HAND_STARTED =
      NotificationType._(22, _omitEnumNames ? '' : 'NEW_HAND_STARTED');
  static const NotificationType PLAYER_ALL_IN =
      NotificationType._(23, _omitEnumNames ? '' : 'PLAYER_ALL_IN');

  /// Connection and stream lifecycle (typed, non-UNKNOWN)
  static const NotificationType NOTIFICATION_STREAM_CONNECTED =
      NotificationType._(
          24, _omitEnumNames ? '' : 'NOTIFICATION_STREAM_CONNECTED');
  static const NotificationType NOTIFICATION_STREAM_DISCONNECTED =
      NotificationType._(
          25, _omitEnumNames ? '' : 'NOTIFICATION_STREAM_DISCONNECTED');
  static const NotificationType GAME_STREAM_CONNECTED =
      NotificationType._(26, _omitEnumNames ? '' : 'GAME_STREAM_CONNECTED');
  static const NotificationType GAME_STREAM_DISCONNECTED =
      NotificationType._(27, _omitEnumNames ? '' : 'GAME_STREAM_DISCONNECTED');
  static const NotificationType ESCROW_FUNDING =
      NotificationType._(28, _omitEnumNames ? '' : 'ESCROW_FUNDING');
  static const NotificationType MESSAGE =
      NotificationType._(29, _omitEnumNames ? '' : 'MESSAGE');
  static const NotificationType SETTLEMENT_BROADCAST =
      NotificationType._(30, _omitEnumNames ? '' : 'SETTLEMENT_BROADCAST');
  static const NotificationType PRESIGN_PENDING =
      NotificationType._(31, _omitEnumNames ? '' : 'PRESIGN_PENDING');
  static const NotificationType GAME_STATE_UPDATED =
      NotificationType._(32, _omitEnumNames ? '' : 'GAME_STATE_UPDATED');
  static const NotificationType PLAYER_LOST =
      NotificationType._(33, _omitEnumNames ? '' : 'PLAYER_LOST');

  static const $core.List<NotificationType> values = <NotificationType>[
    UNKNOWN,
    PLAYER_JOINED,
    PLAYER_LEFT,
    GAME_STARTED,
    GAME_ENDED,
    BET_MADE,
    PLAYER_FOLDED,
    NEW_ROUND,
    SHOWDOWN_RESULT,
    TIP_RECEIVED,
    BALANCE_UPDATED,
    TABLE_CREATED,
    TABLE_REMOVED,
    PLAYER_READY,
    PLAYER_UNREADY,
    ALL_PLAYERS_READY,
    SMALL_BLIND_POSTED,
    BIG_BLIND_POSTED,
    CALL_MADE,
    CHECK_MADE,
    CARDS_SHOWN,
    CARDS_HIDDEN,
    NEW_HAND_STARTED,
    PLAYER_ALL_IN,
    NOTIFICATION_STREAM_CONNECTED,
    NOTIFICATION_STREAM_DISCONNECTED,
    GAME_STREAM_CONNECTED,
    GAME_STREAM_DISCONNECTED,
    ESCROW_FUNDING,
    MESSAGE,
    SETTLEMENT_BROADCAST,
    PRESIGN_PENDING,
    GAME_STATE_UPDATED,
    PLAYER_LOST,
  ];

  static final $core.List<NotificationType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 33);
  static NotificationType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const NotificationType._(super.value, super.name);
}

class HandRank extends $pb.ProtobufEnum {
  static const HandRank HIGH_CARD =
      HandRank._(0, _omitEnumNames ? '' : 'HIGH_CARD');
  static const HandRank PAIR = HandRank._(1, _omitEnumNames ? '' : 'PAIR');
  static const HandRank TWO_PAIR =
      HandRank._(2, _omitEnumNames ? '' : 'TWO_PAIR');
  static const HandRank THREE_OF_A_KIND =
      HandRank._(3, _omitEnumNames ? '' : 'THREE_OF_A_KIND');
  static const HandRank STRAIGHT =
      HandRank._(4, _omitEnumNames ? '' : 'STRAIGHT');
  static const HandRank FLUSH = HandRank._(5, _omitEnumNames ? '' : 'FLUSH');
  static const HandRank FULL_HOUSE =
      HandRank._(6, _omitEnumNames ? '' : 'FULL_HOUSE');
  static const HandRank FOUR_OF_A_KIND =
      HandRank._(7, _omitEnumNames ? '' : 'FOUR_OF_A_KIND');
  static const HandRank STRAIGHT_FLUSH =
      HandRank._(8, _omitEnumNames ? '' : 'STRAIGHT_FLUSH');
  static const HandRank ROYAL_FLUSH =
      HandRank._(9, _omitEnumNames ? '' : 'ROYAL_FLUSH');

  static const $core.List<HandRank> values = <HandRank>[
    HIGH_CARD,
    PAIR,
    TWO_PAIR,
    THREE_OF_A_KIND,
    STRAIGHT,
    FLUSH,
    FULL_HOUSE,
    FOUR_OF_A_KIND,
    STRAIGHT_FLUSH,
    ROYAL_FLUSH,
  ];

  static final $core.List<HandRank?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 9);
  static HandRank? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HandRank._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
