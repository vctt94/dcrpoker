// This is a generated file - do not edit.
//
// Generated from poker.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use gamePhaseDescriptor instead')
const GamePhase$json = {
  '1': 'GamePhase',
  '2': [
    {'1': 'WAITING', '2': 0},
    {'1': 'NEW_HAND_DEALING', '2': 1},
    {'1': 'PRE_FLOP', '2': 2},
    {'1': 'FLOP', '2': 3},
    {'1': 'TURN', '2': 4},
    {'1': 'RIVER', '2': 5},
    {'1': 'SHOWDOWN', '2': 6},
  ],
};

/// Descriptor for `GamePhase`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List gamePhaseDescriptor = $convert.base64Decode(
    'CglHYW1lUGhhc2USCwoHV0FJVElORxAAEhQKEE5FV19IQU5EX0RFQUxJTkcQARIMCghQUkVfRk'
    'xPUBACEggKBEZMT1AQAxIICgRUVVJOEAQSCQoFUklWRVIQBRIMCghTSE9XRE9XThAG');

@$core.Deprecated('Use playerStateDescriptor instead')
const PlayerState$json = {
  '1': 'PlayerState',
  '2': [
    {'1': 'PLAYER_STATE_UNINITIALIZED', '2': 0},
    {'1': 'PLAYER_STATE_AT_TABLE', '2': 1},
    {'1': 'PLAYER_STATE_IN_GAME', '2': 2},
    {'1': 'PLAYER_STATE_ALL_IN', '2': 3},
    {'1': 'PLAYER_STATE_FOLDED', '2': 4},
    {'1': 'PLAYER_STATE_LEFT', '2': 5},
  ],
};

/// Descriptor for `PlayerState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List playerStateDescriptor = $convert.base64Decode(
    'CgtQbGF5ZXJTdGF0ZRIeChpQTEFZRVJfU1RBVEVfVU5JTklUSUFMSVpFRBAAEhkKFVBMQVlFUl'
    '9TVEFURV9BVF9UQUJMRRABEhgKFFBMQVlFUl9TVEFURV9JTl9HQU1FEAISFwoTUExBWUVSX1NU'
    'QVRFX0FMTF9JThADEhcKE1BMQVlFUl9TVEFURV9GT0xERUQQBBIVChFQTEFZRVJfU1RBVEVfTE'
    'VGVBAF');

@$core.Deprecated('Use notificationTypeDescriptor instead')
const NotificationType$json = {
  '1': 'NotificationType',
  '2': [
    {'1': 'UNKNOWN', '2': 0},
    {'1': 'PLAYER_JOINED', '2': 1},
    {'1': 'PLAYER_LEFT', '2': 2},
    {'1': 'GAME_STARTED', '2': 3},
    {'1': 'GAME_ENDED', '2': 4},
    {'1': 'BET_MADE', '2': 5},
    {'1': 'PLAYER_FOLDED', '2': 6},
    {'1': 'NEW_ROUND', '2': 7},
    {'1': 'SHOWDOWN_RESULT', '2': 8},
    {'1': 'TIP_RECEIVED', '2': 9},
    {'1': 'BALANCE_UPDATED', '2': 10},
    {'1': 'TABLE_CREATED', '2': 11},
    {'1': 'TABLE_REMOVED', '2': 12},
    {'1': 'PLAYER_READY', '2': 13},
    {'1': 'PLAYER_UNREADY', '2': 14},
    {'1': 'ALL_PLAYERS_READY', '2': 15},
    {'1': 'SMALL_BLIND_POSTED', '2': 16},
    {'1': 'BIG_BLIND_POSTED', '2': 17},
    {'1': 'CALL_MADE', '2': 18},
    {'1': 'CHECK_MADE', '2': 19},
    {'1': 'CARDS_SHOWN', '2': 20},
    {'1': 'CARDS_HIDDEN', '2': 21},
    {'1': 'NEW_HAND_STARTED', '2': 22},
    {'1': 'PLAYER_ALL_IN', '2': 23},
    {'1': 'NOTIFICATION_STREAM_CONNECTED', '2': 24},
    {'1': 'NOTIFICATION_STREAM_DISCONNECTED', '2': 25},
    {'1': 'GAME_STREAM_CONNECTED', '2': 26},
    {'1': 'GAME_STREAM_DISCONNECTED', '2': 27},
    {'1': 'ESCROW_FUNDING', '2': 28},
  ],
};

/// Descriptor for `NotificationType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List notificationTypeDescriptor = $convert.base64Decode(
    'ChBOb3RpZmljYXRpb25UeXBlEgsKB1VOS05PV04QABIRCg1QTEFZRVJfSk9JTkVEEAESDwoLUE'
    'xBWUVSX0xFRlQQAhIQCgxHQU1FX1NUQVJURUQQAxIOCgpHQU1FX0VOREVEEAQSDAoIQkVUX01B'
    'REUQBRIRCg1QTEFZRVJfRk9MREVEEAYSDQoJTkVXX1JPVU5EEAcSEwoPU0hPV0RPV05fUkVTVU'
    'xUEAgSEAoMVElQX1JFQ0VJVkVEEAkSEwoPQkFMQU5DRV9VUERBVEVEEAoSEQoNVEFCTEVfQ1JF'
    'QVRFRBALEhEKDVRBQkxFX1JFTU9WRUQQDBIQCgxQTEFZRVJfUkVBRFkQDRISCg5QTEFZRVJfVU'
    '5SRUFEWRAOEhUKEUFMTF9QTEFZRVJTX1JFQURZEA8SFgoSU01BTExfQkxJTkRfUE9TVEVEEBAS'
    'FAoQQklHX0JMSU5EX1BPU1RFRBAREg0KCUNBTExfTUFERRASEg4KCkNIRUNLX01BREUQExIPCg'
    'tDQVJEU19TSE9XThAUEhAKDENBUkRTX0hJRERFThAVEhQKEE5FV19IQU5EX1NUQVJURUQQFhIR'
    'Cg1QTEFZRVJfQUxMX0lOEBcSIQodTk9USUZJQ0FUSU9OX1NUUkVBTV9DT05ORUNURUQQGBIkCi'
    'BOT1RJRklDQVRJT05fU1RSRUFNX0RJU0NPTk5FQ1RFRBAZEhkKFUdBTUVfU1RSRUFNX0NPTk5F'
    'Q1RFRBAaEhwKGEdBTUVfU1RSRUFNX0RJU0NPTk5FQ1RFRBAbEhIKDkVTQ1JPV19GVU5ESU5HEB'
    'w=');

@$core.Deprecated('Use handRankDescriptor instead')
const HandRank$json = {
  '1': 'HandRank',
  '2': [
    {'1': 'HIGH_CARD', '2': 0},
    {'1': 'PAIR', '2': 1},
    {'1': 'TWO_PAIR', '2': 2},
    {'1': 'THREE_OF_A_KIND', '2': 3},
    {'1': 'STRAIGHT', '2': 4},
    {'1': 'FLUSH', '2': 5},
    {'1': 'FULL_HOUSE', '2': 6},
    {'1': 'FOUR_OF_A_KIND', '2': 7},
    {'1': 'STRAIGHT_FLUSH', '2': 8},
    {'1': 'ROYAL_FLUSH', '2': 9},
  ],
};

/// Descriptor for `HandRank`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List handRankDescriptor = $convert.base64Decode(
    'CghIYW5kUmFuaxINCglISUdIX0NBUkQQABIICgRQQUlSEAESDAoIVFdPX1BBSVIQAhITCg9USF'
    'JFRV9PRl9BX0tJTkQQAxIMCghTVFJBSUdIVBAEEgkKBUZMVVNIEAUSDgoKRlVMTF9IT1VTRRAG'
    'EhIKDkZPVVJfT0ZfQV9LSU5EEAcSEgoOU1RSQUlHSFRfRkxVU0gQCBIPCgtST1lBTF9GTFVTSB'
    'AJ');

@$core.Deprecated('Use startGameStreamRequestDescriptor instead')
const StartGameStreamRequest$json = {
  '1': 'StartGameStreamRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `StartGameStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startGameStreamRequestDescriptor =
    $convert.base64Decode(
        'ChZTdGFydEdhbWVTdHJlYW1SZXF1ZXN0EhsKCXBsYXllcl9pZBgBIAEoCVIIcGxheWVySWQSGQ'
        'oIdGFibGVfaWQYAiABKAlSB3RhYmxlSWQ=');

@$core.Deprecated('Use gameUpdateDescriptor instead')
const GameUpdate$json = {
  '1': 'GameUpdate',
  '2': [
    {'1': 'table_id', '3': 1, '4': 1, '5': 9, '10': 'tableId'},
    {
      '1': 'phase',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.poker.GamePhase',
      '10': 'phase'
    },
    {
      '1': 'players',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.poker.Player',
      '10': 'players'
    },
    {
      '1': 'community_cards',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.poker.Card',
      '10': 'communityCards'
    },
    {'1': 'pot', '3': 5, '4': 1, '5': 3, '10': 'pot'},
    {'1': 'current_bet', '3': 6, '4': 1, '5': 3, '10': 'currentBet'},
    {'1': 'current_player', '3': 7, '4': 1, '5': 9, '10': 'currentPlayer'},
    {'1': 'min_raise', '3': 8, '4': 1, '5': 3, '10': 'minRaise'},
    {'1': 'max_raise', '3': 9, '4': 1, '5': 3, '10': 'maxRaise'},
    {'1': 'game_started', '3': 10, '4': 1, '5': 8, '10': 'gameStarted'},
    {'1': 'players_required', '3': 11, '4': 1, '5': 5, '10': 'playersRequired'},
    {'1': 'players_joined', '3': 12, '4': 1, '5': 5, '10': 'playersJoined'},
    {'1': 'phase_name', '3': 13, '4': 1, '5': 9, '10': 'phaseName'},
    {
      '1': 'time_bank_seconds',
      '3': 14,
      '4': 1,
      '5': 5,
      '10': 'timeBankSeconds'
    },
    {
      '1': 'turn_deadline_unix_ms',
      '3': 15,
      '4': 1,
      '5': 3,
      '10': 'turnDeadlineUnixMs'
    },
    {'1': 'small_blind', '3': 16, '4': 1, '5': 3, '10': 'smallBlind'},
    {'1': 'big_blind', '3': 17, '4': 1, '5': 3, '10': 'bigBlind'},
  ],
};

/// Descriptor for `GameUpdate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gameUpdateDescriptor = $convert.base64Decode(
    'CgpHYW1lVXBkYXRlEhkKCHRhYmxlX2lkGAEgASgJUgd0YWJsZUlkEiYKBXBoYXNlGAIgASgOMh'
    'AucG9rZXIuR2FtZVBoYXNlUgVwaGFzZRInCgdwbGF5ZXJzGAMgAygLMg0ucG9rZXIuUGxheWVy'
    'UgdwbGF5ZXJzEjQKD2NvbW11bml0eV9jYXJkcxgEIAMoCzILLnBva2VyLkNhcmRSDmNvbW11bm'
    'l0eUNhcmRzEhAKA3BvdBgFIAEoA1IDcG90Eh8KC2N1cnJlbnRfYmV0GAYgASgDUgpjdXJyZW50'
    'QmV0EiUKDmN1cnJlbnRfcGxheWVyGAcgASgJUg1jdXJyZW50UGxheWVyEhsKCW1pbl9yYWlzZR'
    'gIIAEoA1IIbWluUmFpc2USGwoJbWF4X3JhaXNlGAkgASgDUghtYXhSYWlzZRIhCgxnYW1lX3N0'
    'YXJ0ZWQYCiABKAhSC2dhbWVTdGFydGVkEikKEHBsYXllcnNfcmVxdWlyZWQYCyABKAVSD3BsYX'
    'llcnNSZXF1aXJlZBIlCg5wbGF5ZXJzX2pvaW5lZBgMIAEoBVINcGxheWVyc0pvaW5lZBIdCgpw'
    'aGFzZV9uYW1lGA0gASgJUglwaGFzZU5hbWUSKgoRdGltZV9iYW5rX3NlY29uZHMYDiABKAVSD3'
    'RpbWVCYW5rU2Vjb25kcxIxChV0dXJuX2RlYWRsaW5lX3VuaXhfbXMYDyABKANSEnR1cm5EZWFk'
    'bGluZVVuaXhNcxIfCgtzbWFsbF9ibGluZBgQIAEoA1IKc21hbGxCbGluZBIbCgliaWdfYmxpbm'
    'QYESABKANSCGJpZ0JsaW5k');

@$core.Deprecated('Use makeBetRequestDescriptor instead')
const MakeBetRequest$json = {
  '1': 'MakeBetRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'amount', '3': 3, '4': 1, '5': 3, '10': 'amount'},
  ],
};

/// Descriptor for `MakeBetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List makeBetRequestDescriptor = $convert.base64Decode(
    'Cg5NYWtlQmV0UmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEhkKCHRhYmxlX2'
    'lkGAIgASgJUgd0YWJsZUlkEhYKBmFtb3VudBgDIAEoA1IGYW1vdW50');

@$core.Deprecated('Use makeBetResponseDescriptor instead')
const MakeBetResponse$json = {
  '1': 'MakeBetResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'new_balance', '3': 3, '4': 1, '5': 3, '10': 'newBalance'},
  ],
};

/// Descriptor for `MakeBetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List makeBetResponseDescriptor = $convert.base64Decode(
    'Cg9NYWtlQmV0UmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCgdtZXNzYWdlGA'
    'IgASgJUgdtZXNzYWdlEh8KC25ld19iYWxhbmNlGAMgASgDUgpuZXdCYWxhbmNl');

@$core.Deprecated('Use foldBetRequestDescriptor instead')
const FoldBetRequest$json = {
  '1': 'FoldBetRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `FoldBetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List foldBetRequestDescriptor = $convert.base64Decode(
    'Cg5Gb2xkQmV0UmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEhkKCHRhYmxlX2'
    'lkGAIgASgJUgd0YWJsZUlk');

@$core.Deprecated('Use foldBetResponseDescriptor instead')
const FoldBetResponse$json = {
  '1': 'FoldBetResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `FoldBetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List foldBetResponseDescriptor = $convert.base64Decode(
    'Cg9Gb2xkQmV0UmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCgdtZXNzYWdlGA'
    'IgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use checkBetRequestDescriptor instead')
const CheckBetRequest$json = {
  '1': 'CheckBetRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `CheckBetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List checkBetRequestDescriptor = $convert.base64Decode(
    'Cg9DaGVja0JldFJlcXVlc3QSGwoJcGxheWVyX2lkGAEgASgJUghwbGF5ZXJJZBIZCgh0YWJsZV'
    '9pZBgCIAEoCVIHdGFibGVJZA==');

@$core.Deprecated('Use checkBetResponseDescriptor instead')
const CheckBetResponse$json = {
  '1': 'CheckBetResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `CheckBetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List checkBetResponseDescriptor = $convert.base64Decode(
    'ChBDaGVja0JldFJlc3BvbnNlEhgKB3N1Y2Nlc3MYASABKAhSB3N1Y2Nlc3MSGAoHbWVzc2FnZR'
    'gCIAEoCVIHbWVzc2FnZQ==');

@$core.Deprecated('Use callBetRequestDescriptor instead')
const CallBetRequest$json = {
  '1': 'CallBetRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `CallBetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List callBetRequestDescriptor = $convert.base64Decode(
    'Cg5DYWxsQmV0UmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEhkKCHRhYmxlX2'
    'lkGAIgASgJUgd0YWJsZUlk');

@$core.Deprecated('Use callBetResponseDescriptor instead')
const CallBetResponse$json = {
  '1': 'CallBetResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `CallBetResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List callBetResponseDescriptor = $convert.base64Decode(
    'Cg9DYWxsQmV0UmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCgdtZXNzYWdlGA'
    'IgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use getGameStateRequestDescriptor instead')
const GetGameStateRequest$json = {
  '1': 'GetGameStateRequest',
  '2': [
    {'1': 'table_id', '3': 1, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'player_id', '3': 2, '4': 1, '5': 9, '10': 'playerId'},
  ],
};

/// Descriptor for `GetGameStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getGameStateRequestDescriptor = $convert.base64Decode(
    'ChNHZXRHYW1lU3RhdGVSZXF1ZXN0EhkKCHRhYmxlX2lkGAEgASgJUgd0YWJsZUlkEhsKCXBsYX'
    'llcl9pZBgCIAEoCVIIcGxheWVySWQ=');

@$core.Deprecated('Use getGameStateResponseDescriptor instead')
const GetGameStateResponse$json = {
  '1': 'GetGameStateResponse',
  '2': [
    {
      '1': 'game_state',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.poker.GameUpdate',
      '10': 'gameState'
    },
  ],
};

/// Descriptor for `GetGameStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getGameStateResponseDescriptor = $convert.base64Decode(
    'ChRHZXRHYW1lU3RhdGVSZXNwb25zZRIwCgpnYW1lX3N0YXRlGAEgASgLMhEucG9rZXIuR2FtZV'
    'VwZGF0ZVIJZ2FtZVN0YXRl');

@$core.Deprecated('Use evaluateHandRequestDescriptor instead')
const EvaluateHandRequest$json = {
  '1': 'EvaluateHandRequest',
  '2': [
    {'1': 'cards', '3': 1, '4': 3, '5': 11, '6': '.poker.Card', '10': 'cards'},
  ],
};

/// Descriptor for `EvaluateHandRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List evaluateHandRequestDescriptor = $convert.base64Decode(
    'ChNFdmFsdWF0ZUhhbmRSZXF1ZXN0EiEKBWNhcmRzGAEgAygLMgsucG9rZXIuQ2FyZFIFY2FyZH'
    'M=');

@$core.Deprecated('Use evaluateHandResponseDescriptor instead')
const EvaluateHandResponse$json = {
  '1': 'EvaluateHandResponse',
  '2': [
    {
      '1': 'rank',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.poker.HandRank',
      '10': 'rank'
    },
    {'1': 'description', '3': 2, '4': 1, '5': 9, '10': 'description'},
    {
      '1': 'best_hand',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.poker.Card',
      '10': 'bestHand'
    },
  ],
};

/// Descriptor for `EvaluateHandResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List evaluateHandResponseDescriptor = $convert.base64Decode(
    'ChRFdmFsdWF0ZUhhbmRSZXNwb25zZRIjCgRyYW5rGAEgASgOMg8ucG9rZXIuSGFuZFJhbmtSBH'
    'JhbmsSIAoLZGVzY3JpcHRpb24YAiABKAlSC2Rlc2NyaXB0aW9uEigKCWJlc3RfaGFuZBgDIAMo'
    'CzILLnBva2VyLkNhcmRSCGJlc3RIYW5k');

@$core.Deprecated('Use getLastWinnersRequestDescriptor instead')
const GetLastWinnersRequest$json = {
  '1': 'GetLastWinnersRequest',
  '2': [
    {'1': 'table_id', '3': 1, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `GetLastWinnersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLastWinnersRequestDescriptor =
    $convert.base64Decode(
        'ChVHZXRMYXN0V2lubmVyc1JlcXVlc3QSGQoIdGFibGVfaWQYASABKAlSB3RhYmxlSWQ=');

@$core.Deprecated('Use getLastWinnersResponseDescriptor instead')
const GetLastWinnersResponse$json = {
  '1': 'GetLastWinnersResponse',
  '2': [
    {
      '1': 'winners',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.poker.Winner',
      '10': 'winners'
    },
  ],
};

/// Descriptor for `GetLastWinnersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLastWinnersResponseDescriptor =
    $convert.base64Decode(
        'ChZHZXRMYXN0V2lubmVyc1Jlc3BvbnNlEicKB3dpbm5lcnMYASADKAsyDS5wb2tlci5XaW5uZX'
        'JSB3dpbm5lcnM=');

@$core.Deprecated('Use winnerDescriptor instead')
const Winner$json = {
  '1': 'Winner',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {
      '1': 'hand_rank',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.poker.HandRank',
      '10': 'handRank'
    },
    {
      '1': 'best_hand',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.poker.Card',
      '10': 'bestHand'
    },
    {'1': 'winnings', '3': 4, '4': 1, '5': 3, '10': 'winnings'},
  ],
};

/// Descriptor for `Winner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List winnerDescriptor = $convert.base64Decode(
    'CgZXaW5uZXISGwoJcGxheWVyX2lkGAEgASgJUghwbGF5ZXJJZBIsCgloYW5kX3JhbmsYAiABKA'
    '4yDy5wb2tlci5IYW5kUmFua1IIaGFuZFJhbmsSKAoJYmVzdF9oYW5kGAMgAygLMgsucG9rZXIu'
    'Q2FyZFIIYmVzdEhhbmQSGgoId2lubmluZ3MYBCABKANSCHdpbm5pbmdz');

@$core.Deprecated('Use createTableRequestDescriptor instead')
const CreateTableRequest$json = {
  '1': 'CreateTableRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'small_blind', '3': 2, '4': 1, '5': 3, '10': 'smallBlind'},
    {'1': 'big_blind', '3': 3, '4': 1, '5': 3, '10': 'bigBlind'},
    {'1': 'max_players', '3': 4, '4': 1, '5': 5, '10': 'maxPlayers'},
    {'1': 'min_players', '3': 5, '4': 1, '5': 5, '10': 'minPlayers'},
    {'1': 'min_balance', '3': 6, '4': 1, '5': 3, '10': 'minBalance'},
    {'1': 'buy_in', '3': 7, '4': 1, '5': 3, '10': 'buyIn'},
    {'1': 'starting_chips', '3': 8, '4': 1, '5': 3, '10': 'startingChips'},
    {'1': 'time_bank_seconds', '3': 9, '4': 1, '5': 5, '10': 'timeBankSeconds'},
    {'1': 'auto_start_ms', '3': 10, '4': 1, '5': 5, '10': 'autoStartMs'},
    {'1': 'auto_advance_ms', '3': 11, '4': 1, '5': 5, '10': 'autoAdvanceMs'},
  ],
};

/// Descriptor for `CreateTableRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createTableRequestDescriptor = $convert.base64Decode(
    'ChJDcmVhdGVUYWJsZVJlcXVlc3QSGwoJcGxheWVyX2lkGAEgASgJUghwbGF5ZXJJZBIfCgtzbW'
    'FsbF9ibGluZBgCIAEoA1IKc21hbGxCbGluZBIbCgliaWdfYmxpbmQYAyABKANSCGJpZ0JsaW5k'
    'Eh8KC21heF9wbGF5ZXJzGAQgASgFUgptYXhQbGF5ZXJzEh8KC21pbl9wbGF5ZXJzGAUgASgFUg'
    'ptaW5QbGF5ZXJzEh8KC21pbl9iYWxhbmNlGAYgASgDUgptaW5CYWxhbmNlEhUKBmJ1eV9pbhgH'
    'IAEoA1IFYnV5SW4SJQoOc3RhcnRpbmdfY2hpcHMYCCABKANSDXN0YXJ0aW5nQ2hpcHMSKgoRdG'
    'ltZV9iYW5rX3NlY29uZHMYCSABKAVSD3RpbWVCYW5rU2Vjb25kcxIiCg1hdXRvX3N0YXJ0X21z'
    'GAogASgFUgthdXRvU3RhcnRNcxImCg9hdXRvX2FkdmFuY2VfbXMYCyABKAVSDWF1dG9BZHZhbm'
    'NlTXM=');

@$core.Deprecated('Use createTableResponseDescriptor instead')
const CreateTableResponse$json = {
  '1': 'CreateTableResponse',
  '2': [
    {'1': 'table_id', '3': 1, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `CreateTableResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createTableResponseDescriptor = $convert.base64Decode(
    'ChNDcmVhdGVUYWJsZVJlc3BvbnNlEhkKCHRhYmxlX2lkGAEgASgJUgd0YWJsZUlkEhgKB21lc3'
    'NhZ2UYAiABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use joinTableRequestDescriptor instead')
const JoinTableRequest$json = {
  '1': 'JoinTableRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `JoinTableRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List joinTableRequestDescriptor = $convert.base64Decode(
    'ChBKb2luVGFibGVSZXF1ZXN0EhsKCXBsYXllcl9pZBgBIAEoCVIIcGxheWVySWQSGQoIdGFibG'
    'VfaWQYAiABKAlSB3RhYmxlSWQ=');

@$core.Deprecated('Use joinTableResponseDescriptor instead')
const JoinTableResponse$json = {
  '1': 'JoinTableResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `JoinTableResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List joinTableResponseDescriptor = $convert.base64Decode(
    'ChFKb2luVGFibGVSZXNwb25zZRIYCgdzdWNjZXNzGAEgASgIUgdzdWNjZXNzEhgKB21lc3NhZ2'
    'UYAiABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use leaveTableRequestDescriptor instead')
const LeaveTableRequest$json = {
  '1': 'LeaveTableRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `LeaveTableRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List leaveTableRequestDescriptor = $convert.base64Decode(
    'ChFMZWF2ZVRhYmxlUmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEhkKCHRhYm'
    'xlX2lkGAIgASgJUgd0YWJsZUlk');

@$core.Deprecated('Use leaveTableResponseDescriptor instead')
const LeaveTableResponse$json = {
  '1': 'LeaveTableResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `LeaveTableResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List leaveTableResponseDescriptor = $convert.base64Decode(
    'ChJMZWF2ZVRhYmxlUmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCgdtZXNzYW'
    'dlGAIgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use getTablesRequestDescriptor instead')
const GetTablesRequest$json = {
  '1': 'GetTablesRequest',
};

/// Descriptor for `GetTablesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTablesRequestDescriptor =
    $convert.base64Decode('ChBHZXRUYWJsZXNSZXF1ZXN0');

@$core.Deprecated('Use getTablesResponseDescriptor instead')
const GetTablesResponse$json = {
  '1': 'GetTablesResponse',
  '2': [
    {
      '1': 'tables',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.poker.Table',
      '10': 'tables'
    },
  ],
};

/// Descriptor for `GetTablesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTablesResponseDescriptor = $convert.base64Decode(
    'ChFHZXRUYWJsZXNSZXNwb25zZRIkCgZ0YWJsZXMYASADKAsyDC5wb2tlci5UYWJsZVIGdGFibG'
    'Vz');

@$core.Deprecated('Use tableDescriptor instead')
const Table$json = {
  '1': 'Table',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'host_id', '3': 2, '4': 1, '5': 9, '10': 'hostId'},
    {
      '1': 'players',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.poker.Player',
      '10': 'players'
    },
    {'1': 'small_blind', '3': 4, '4': 1, '5': 3, '10': 'smallBlind'},
    {'1': 'big_blind', '3': 5, '4': 1, '5': 3, '10': 'bigBlind'},
    {'1': 'max_players', '3': 6, '4': 1, '5': 5, '10': 'maxPlayers'},
    {'1': 'min_players', '3': 7, '4': 1, '5': 5, '10': 'minPlayers'},
    {'1': 'current_players', '3': 8, '4': 1, '5': 5, '10': 'currentPlayers'},
    {'1': 'min_balance', '3': 9, '4': 1, '5': 3, '10': 'minBalance'},
    {'1': 'buy_in', '3': 10, '4': 1, '5': 3, '10': 'buyIn'},
    {
      '1': 'phase',
      '3': 11,
      '4': 1,
      '5': 14,
      '6': '.poker.GamePhase',
      '10': 'phase'
    },
    {'1': 'game_started', '3': 12, '4': 1, '5': 8, '10': 'gameStarted'},
    {
      '1': 'all_players_ready',
      '3': 13,
      '4': 1,
      '5': 8,
      '10': 'allPlayersReady'
    },
  ],
};

/// Descriptor for `Table`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List tableDescriptor = $convert.base64Decode(
    'CgVUYWJsZRIOCgJpZBgBIAEoCVICaWQSFwoHaG9zdF9pZBgCIAEoCVIGaG9zdElkEicKB3BsYX'
    'llcnMYAyADKAsyDS5wb2tlci5QbGF5ZXJSB3BsYXllcnMSHwoLc21hbGxfYmxpbmQYBCABKANS'
    'CnNtYWxsQmxpbmQSGwoJYmlnX2JsaW5kGAUgASgDUghiaWdCbGluZBIfCgttYXhfcGxheWVycx'
    'gGIAEoBVIKbWF4UGxheWVycxIfCgttaW5fcGxheWVycxgHIAEoBVIKbWluUGxheWVycxInCg9j'
    'dXJyZW50X3BsYXllcnMYCCABKAVSDmN1cnJlbnRQbGF5ZXJzEh8KC21pbl9iYWxhbmNlGAkgAS'
    'gDUgptaW5CYWxhbmNlEhUKBmJ1eV9pbhgKIAEoA1IFYnV5SW4SJgoFcGhhc2UYCyABKA4yEC5w'
    'b2tlci5HYW1lUGhhc2VSBXBoYXNlEiEKDGdhbWVfc3RhcnRlZBgMIAEoCFILZ2FtZVN0YXJ0ZW'
    'QSKgoRYWxsX3BsYXllcnNfcmVhZHkYDSABKAhSD2FsbFBsYXllcnNSZWFkeQ==');

@$core.Deprecated('Use getBalanceRequestDescriptor instead')
const GetBalanceRequest$json = {
  '1': 'GetBalanceRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
  ],
};

/// Descriptor for `GetBalanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getBalanceRequestDescriptor = $convert.base64Decode(
    'ChFHZXRCYWxhbmNlUmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklk');

@$core.Deprecated('Use getBalanceResponseDescriptor instead')
const GetBalanceResponse$json = {
  '1': 'GetBalanceResponse',
  '2': [
    {'1': 'balance', '3': 1, '4': 1, '5': 3, '10': 'balance'},
  ],
};

/// Descriptor for `GetBalanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getBalanceResponseDescriptor =
    $convert.base64Decode(
        'ChJHZXRCYWxhbmNlUmVzcG9uc2USGAoHYmFsYW5jZRgBIAEoA1IHYmFsYW5jZQ==');

@$core.Deprecated('Use updateBalanceRequestDescriptor instead')
const UpdateBalanceRequest$json = {
  '1': 'UpdateBalanceRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'amount', '3': 2, '4': 1, '5': 3, '10': 'amount'},
    {'1': 'description', '3': 3, '4': 1, '5': 9, '10': 'description'},
  ],
};

/// Descriptor for `UpdateBalanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateBalanceRequestDescriptor = $convert.base64Decode(
    'ChRVcGRhdGVCYWxhbmNlUmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEhYKBm'
    'Ftb3VudBgCIAEoA1IGYW1vdW50EiAKC2Rlc2NyaXB0aW9uGAMgASgJUgtkZXNjcmlwdGlvbg==');

@$core.Deprecated('Use updateBalanceResponseDescriptor instead')
const UpdateBalanceResponse$json = {
  '1': 'UpdateBalanceResponse',
  '2': [
    {'1': 'new_balance', '3': 1, '4': 1, '5': 3, '10': 'newBalance'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `UpdateBalanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateBalanceResponseDescriptor = $convert.base64Decode(
    'ChVVcGRhdGVCYWxhbmNlUmVzcG9uc2USHwoLbmV3X2JhbGFuY2UYASABKANSCm5ld0JhbGFuY2'
    'USGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZQ==');

@$core.Deprecated('Use processTipRequestDescriptor instead')
const ProcessTipRequest$json = {
  '1': 'ProcessTipRequest',
  '2': [
    {'1': 'from_player_id', '3': 1, '4': 1, '5': 9, '10': 'fromPlayerId'},
    {'1': 'to_player_id', '3': 2, '4': 1, '5': 9, '10': 'toPlayerId'},
    {'1': 'amount', '3': 3, '4': 1, '5': 3, '10': 'amount'},
    {'1': 'message', '3': 4, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ProcessTipRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List processTipRequestDescriptor = $convert.base64Decode(
    'ChFQcm9jZXNzVGlwUmVxdWVzdBIkCg5mcm9tX3BsYXllcl9pZBgBIAEoCVIMZnJvbVBsYXllck'
    'lkEiAKDHRvX3BsYXllcl9pZBgCIAEoCVIKdG9QbGF5ZXJJZBIWCgZhbW91bnQYAyABKANSBmFt'
    'b3VudBIYCgdtZXNzYWdlGAQgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use processTipResponseDescriptor instead')
const ProcessTipResponse$json = {
  '1': 'ProcessTipResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'new_balance', '3': 3, '4': 1, '5': 3, '10': 'newBalance'},
  ],
};

/// Descriptor for `ProcessTipResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List processTipResponseDescriptor = $convert.base64Decode(
    'ChJQcm9jZXNzVGlwUmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCgdtZXNzYW'
    'dlGAIgASgJUgdtZXNzYWdlEh8KC25ld19iYWxhbmNlGAMgASgDUgpuZXdCYWxhbmNl');

@$core.Deprecated('Use startNotificationStreamRequestDescriptor instead')
const StartNotificationStreamRequest$json = {
  '1': 'StartNotificationStreamRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
  ],
};

/// Descriptor for `StartNotificationStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startNotificationStreamRequestDescriptor =
    $convert.base64Decode(
        'Ch5TdGFydE5vdGlmaWNhdGlvblN0cmVhbVJlcXVlc3QSGwoJcGxheWVyX2lkGAEgASgJUghwbG'
        'F5ZXJJZA==');

@$core.Deprecated('Use notificationDescriptor instead')
const Notification$json = {
  '1': 'Notification',
  '2': [
    {
      '1': 'type',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.poker.NotificationType',
      '10': 'type'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'table_id', '3': 3, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'player_id', '3': 4, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'amount', '3': 5, '4': 1, '5': 3, '10': 'amount'},
    {'1': 'cards', '3': 6, '4': 3, '5': 11, '6': '.poker.Card', '10': 'cards'},
    {
      '1': 'hand_rank',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.poker.HandRank',
      '10': 'handRank'
    },
    {'1': 'new_balance', '3': 8, '4': 1, '5': 3, '10': 'newBalance'},
    {'1': 'table', '3': 9, '4': 1, '5': 11, '6': '.poker.Table', '10': 'table'},
    {'1': 'ready', '3': 10, '4': 1, '5': 8, '10': 'ready'},
    {'1': 'started', '3': 11, '4': 1, '5': 8, '10': 'started'},
    {
      '1': 'game_ready_to_play',
      '3': 12,
      '4': 1,
      '5': 8,
      '10': 'gameReadyToPlay'
    },
    {'1': 'countdown', '3': 13, '4': 1, '5': 5, '10': 'countdown'},
    {
      '1': 'winners',
      '3': 14,
      '4': 3,
      '5': 11,
      '6': '.poker.Winner',
      '10': 'winners'
    },
    {
      '1': 'showdown',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.poker.Showdown',
      '10': 'showdown'
    },
  ],
};

/// Descriptor for `Notification`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List notificationDescriptor = $convert.base64Decode(
    'CgxOb3RpZmljYXRpb24SKwoEdHlwZRgBIAEoDjIXLnBva2VyLk5vdGlmaWNhdGlvblR5cGVSBH'
    'R5cGUSGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZRIZCgh0YWJsZV9pZBgDIAEoCVIHdGFibGVJ'
    'ZBIbCglwbGF5ZXJfaWQYBCABKAlSCHBsYXllcklkEhYKBmFtb3VudBgFIAEoA1IGYW1vdW50Ei'
    'EKBWNhcmRzGAYgAygLMgsucG9rZXIuQ2FyZFIFY2FyZHMSLAoJaGFuZF9yYW5rGAcgASgOMg8u'
    'cG9rZXIuSGFuZFJhbmtSCGhhbmRSYW5rEh8KC25ld19iYWxhbmNlGAggASgDUgpuZXdCYWxhbm'
    'NlEiIKBXRhYmxlGAkgASgLMgwucG9rZXIuVGFibGVSBXRhYmxlEhQKBXJlYWR5GAogASgIUgVy'
    'ZWFkeRIYCgdzdGFydGVkGAsgASgIUgdzdGFydGVkEisKEmdhbWVfcmVhZHlfdG9fcGxheRgMIA'
    'EoCFIPZ2FtZVJlYWR5VG9QbGF5EhwKCWNvdW50ZG93bhgNIAEoBVIJY291bnRkb3duEicKB3dp'
    'bm5lcnMYDiADKAsyDS5wb2tlci5XaW5uZXJSB3dpbm5lcnMSKwoIc2hvd2Rvd24YDyABKAsyDy'
    '5wb2tlci5TaG93ZG93blIIc2hvd2Rvd24=');

@$core.Deprecated('Use showdownDescriptor instead')
const Showdown$json = {
  '1': 'Showdown',
  '2': [
    {
      '1': 'winners',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.poker.Winner',
      '10': 'winners'
    },
    {'1': 'pot', '3': 2, '4': 1, '5': 3, '10': 'pot'},
  ],
};

/// Descriptor for `Showdown`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List showdownDescriptor = $convert.base64Decode(
    'CghTaG93ZG93bhInCgd3aW5uZXJzGAEgAygLMg0ucG9rZXIuV2lubmVyUgd3aW5uZXJzEhAKA3'
    'BvdBgCIAEoA1IDcG90');

@$core.Deprecated('Use playerDescriptor instead')
const Player$json = {
  '1': 'Player',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'balance', '3': 3, '4': 1, '5': 3, '10': 'balance'},
    {'1': 'hand', '3': 4, '4': 3, '5': 11, '6': '.poker.Card', '10': 'hand'},
    {'1': 'current_bet', '3': 5, '4': 1, '5': 3, '10': 'currentBet'},
    {'1': 'folded', '3': 6, '4': 1, '5': 8, '10': 'folded'},
    {'1': 'is_turn', '3': 7, '4': 1, '5': 8, '10': 'isTurn'},
    {'1': 'is_all_in', '3': 8, '4': 1, '5': 8, '10': 'isAllIn'},
    {'1': 'is_dealer', '3': 9, '4': 1, '5': 8, '10': 'isDealer'},
    {'1': 'is_ready', '3': 10, '4': 1, '5': 8, '10': 'isReady'},
    {'1': 'hand_description', '3': 11, '4': 1, '5': 9, '10': 'handDescription'},
    {
      '1': 'player_state',
      '3': 12,
      '4': 1,
      '5': 14,
      '6': '.poker.PlayerState',
      '10': 'playerState'
    },
    {'1': 'is_small_blind', '3': 13, '4': 1, '5': 8, '10': 'isSmallBlind'},
    {'1': 'is_big_blind', '3': 14, '4': 1, '5': 8, '10': 'isBigBlind'},
    {'1': 'is_disconnected', '3': 15, '4': 1, '5': 8, '10': 'isDisconnected'},
    {'1': 'escrow_id', '3': 16, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'escrow_ready', '3': 17, '4': 1, '5': 8, '10': 'escrowReady'},
  ],
};

/// Descriptor for `Player`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List playerDescriptor = $convert.base64Decode(
    'CgZQbGF5ZXISDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbWUSGAoHYmFsYW5jZR'
    'gDIAEoA1IHYmFsYW5jZRIfCgRoYW5kGAQgAygLMgsucG9rZXIuQ2FyZFIEaGFuZBIfCgtjdXJy'
    'ZW50X2JldBgFIAEoA1IKY3VycmVudEJldBIWCgZmb2xkZWQYBiABKAhSBmZvbGRlZBIXCgdpc1'
    '90dXJuGAcgASgIUgZpc1R1cm4SGgoJaXNfYWxsX2luGAggASgIUgdpc0FsbEluEhsKCWlzX2Rl'
    'YWxlchgJIAEoCFIIaXNEZWFsZXISGQoIaXNfcmVhZHkYCiABKAhSB2lzUmVhZHkSKQoQaGFuZF'
    '9kZXNjcmlwdGlvbhgLIAEoCVIPaGFuZERlc2NyaXB0aW9uEjUKDHBsYXllcl9zdGF0ZRgMIAEo'
    'DjISLnBva2VyLlBsYXllclN0YXRlUgtwbGF5ZXJTdGF0ZRIkCg5pc19zbWFsbF9ibGluZBgNIA'
    'EoCFIMaXNTbWFsbEJsaW5kEiAKDGlzX2JpZ19ibGluZBgOIAEoCFIKaXNCaWdCbGluZBInCg9p'
    'c19kaXNjb25uZWN0ZWQYDyABKAhSDmlzRGlzY29ubmVjdGVkEhsKCWVzY3Jvd19pZBgQIAEoCV'
    'IIZXNjcm93SWQSIQoMZXNjcm93X3JlYWR5GBEgASgIUgtlc2Nyb3dSZWFkeQ==');

@$core.Deprecated('Use cardDescriptor instead')
const Card$json = {
  '1': 'Card',
  '2': [
    {'1': 'suit', '3': 1, '4': 1, '5': 9, '10': 'suit'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
};

/// Descriptor for `Card`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cardDescriptor = $convert.base64Decode(
    'CgRDYXJkEhIKBHN1aXQYASABKAlSBHN1aXQSFAoFdmFsdWUYAiABKAlSBXZhbHVl');

@$core.Deprecated('Use setPlayerReadyRequestDescriptor instead')
const SetPlayerReadyRequest$json = {
  '1': 'SetPlayerReadyRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `SetPlayerReadyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPlayerReadyRequestDescriptor = $convert.base64Decode(
    'ChVTZXRQbGF5ZXJSZWFkeVJlcXVlc3QSGwoJcGxheWVyX2lkGAEgASgJUghwbGF5ZXJJZBIZCg'
    'h0YWJsZV9pZBgCIAEoCVIHdGFibGVJZA==');

@$core.Deprecated('Use setPlayerReadyResponseDescriptor instead')
const SetPlayerReadyResponse$json = {
  '1': 'SetPlayerReadyResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'all_players_ready', '3': 3, '4': 1, '5': 8, '10': 'allPlayersReady'},
  ],
};

/// Descriptor for `SetPlayerReadyResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPlayerReadyResponseDescriptor = $convert.base64Decode(
    'ChZTZXRQbGF5ZXJSZWFkeVJlc3BvbnNlEhgKB3N1Y2Nlc3MYASABKAhSB3N1Y2Nlc3MSGAoHbW'
    'Vzc2FnZRgCIAEoCVIHbWVzc2FnZRIqChFhbGxfcGxheWVyc19yZWFkeRgDIAEoCFIPYWxsUGxh'
    'eWVyc1JlYWR5');

@$core.Deprecated('Use setPlayerUnreadyRequestDescriptor instead')
const SetPlayerUnreadyRequest$json = {
  '1': 'SetPlayerUnreadyRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `SetPlayerUnreadyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPlayerUnreadyRequestDescriptor =
    $convert.base64Decode(
        'ChdTZXRQbGF5ZXJVbnJlYWR5UmVxdWVzdBIbCglwbGF5ZXJfaWQYASABKAlSCHBsYXllcklkEh'
        'kKCHRhYmxlX2lkGAIgASgJUgd0YWJsZUlk');

@$core.Deprecated('Use setPlayerUnreadyResponseDescriptor instead')
const SetPlayerUnreadyResponse$json = {
  '1': 'SetPlayerUnreadyResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `SetPlayerUnreadyResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPlayerUnreadyResponseDescriptor =
    $convert.base64Decode(
        'ChhTZXRQbGF5ZXJVbnJlYWR5UmVzcG9uc2USGAoHc3VjY2VzcxgBIAEoCFIHc3VjY2VzcxIYCg'
        'dtZXNzYWdlGAIgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use getPlayerCurrentTableRequestDescriptor instead')
const GetPlayerCurrentTableRequest$json = {
  '1': 'GetPlayerCurrentTableRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
  ],
};

/// Descriptor for `GetPlayerCurrentTableRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPlayerCurrentTableRequestDescriptor =
    $convert.base64Decode(
        'ChxHZXRQbGF5ZXJDdXJyZW50VGFibGVSZXF1ZXN0EhsKCXBsYXllcl9pZBgBIAEoCVIIcGxheW'
        'VySWQ=');

@$core.Deprecated('Use getPlayerCurrentTableResponseDescriptor instead')
const GetPlayerCurrentTableResponse$json = {
  '1': 'GetPlayerCurrentTableResponse',
  '2': [
    {'1': 'table_id', '3': 1, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `GetPlayerCurrentTableResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPlayerCurrentTableResponseDescriptor =
    $convert.base64Decode(
        'Ch1HZXRQbGF5ZXJDdXJyZW50VGFibGVSZXNwb25zZRIZCgh0YWJsZV9pZBgBIAEoCVIHdGFibG'
        'VJZA==');

@$core.Deprecated('Use showCardsRequestDescriptor instead')
const ShowCardsRequest$json = {
  '1': 'ShowCardsRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `ShowCardsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List showCardsRequestDescriptor = $convert.base64Decode(
    'ChBTaG93Q2FyZHNSZXF1ZXN0EhsKCXBsYXllcl9pZBgBIAEoCVIIcGxheWVySWQSGQoIdGFibG'
    'VfaWQYAiABKAlSB3RhYmxlSWQ=');

@$core.Deprecated('Use showCardsResponseDescriptor instead')
const ShowCardsResponse$json = {
  '1': 'ShowCardsResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ShowCardsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List showCardsResponseDescriptor = $convert.base64Decode(
    'ChFTaG93Q2FyZHNSZXNwb25zZRIYCgdzdWNjZXNzGAEgASgIUgdzdWNjZXNzEhgKB21lc3NhZ2'
    'UYAiABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use hideCardsRequestDescriptor instead')
const HideCardsRequest$json = {
  '1': 'HideCardsRequest',
  '2': [
    {'1': 'player_id', '3': 1, '4': 1, '5': 9, '10': 'playerId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
  ],
};

/// Descriptor for `HideCardsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List hideCardsRequestDescriptor = $convert.base64Decode(
    'ChBIaWRlQ2FyZHNSZXF1ZXN0EhsKCXBsYXllcl9pZBgBIAEoCVIIcGxheWVySWQSGQoIdGFibG'
    'VfaWQYAiABKAlSB3RhYmxlSWQ=');

@$core.Deprecated('Use hideCardsResponseDescriptor instead')
const HideCardsResponse$json = {
  '1': 'HideCardsResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `HideCardsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List hideCardsResponseDescriptor = $convert.base64Decode(
    'ChFIaWRlQ2FyZHNSZXNwb25zZRIYCgdzdWNjZXNzGAEgASgIUgdzdWNjZXNzEhgKB21lc3NhZ2'
    'UYAiABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use registerRequestDescriptor instead')
const RegisterRequest$json = {
  '1': 'RegisterRequest',
  '2': [
    {'1': 'nickname', '3': 1, '4': 1, '5': 9, '10': 'nickname'},
    {'1': 'user_id', '3': 2, '4': 1, '5': 9, '10': 'userId'},
  ],
};

/// Descriptor for `RegisterRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerRequestDescriptor = $convert.base64Decode(
    'Cg9SZWdpc3RlclJlcXVlc3QSGgoIbmlja25hbWUYASABKAlSCG5pY2tuYW1lEhcKB3VzZXJfaW'
    'QYAiABKAlSBnVzZXJJZA==');

@$core.Deprecated('Use registerResponseDescriptor instead')
const RegisterResponse$json = {
  '1': 'RegisterResponse',
  '2': [
    {'1': 'ok', '3': 1, '4': 1, '5': 8, '10': 'ok'},
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `RegisterResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerResponseDescriptor = $convert.base64Decode(
    'ChBSZWdpc3RlclJlc3BvbnNlEg4KAm9rGAEgASgIUgJvaxIUCgVlcnJvchgCIAEoCVIFZXJyb3'
    'I=');

@$core.Deprecated('Use loginRequestDescriptor instead')
const LoginRequest$json = {
  '1': 'LoginRequest',
  '2': [
    {'1': 'nickname', '3': 1, '4': 1, '5': 9, '10': 'nickname'},
    {'1': 'user_id', '3': 2, '4': 1, '5': 9, '10': 'userId'},
    {'1': 'address', '3': 3, '4': 1, '5': 9, '10': 'address'},
    {'1': 'signature', '3': 4, '4': 1, '5': 9, '10': 'signature'},
    {'1': 'code', '3': 5, '4': 1, '5': 9, '10': 'code'},
  ],
};

/// Descriptor for `LoginRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List loginRequestDescriptor = $convert.base64Decode(
    'CgxMb2dpblJlcXVlc3QSGgoIbmlja25hbWUYASABKAlSCG5pY2tuYW1lEhcKB3VzZXJfaWQYAi'
    'ABKAlSBnVzZXJJZBIYCgdhZGRyZXNzGAMgASgJUgdhZGRyZXNzEhwKCXNpZ25hdHVyZRgEIAEo'
    'CVIJc2lnbmF0dXJlEhIKBGNvZGUYBSABKAlSBGNvZGU=');

@$core.Deprecated('Use requestLoginCodeRequestDescriptor instead')
const RequestLoginCodeRequest$json = {
  '1': 'RequestLoginCodeRequest',
  '2': [
    {'1': 'user_id', '3': 1, '4': 1, '5': 9, '10': 'userId'},
  ],
};

/// Descriptor for `RequestLoginCodeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestLoginCodeRequestDescriptor =
    $convert.base64Decode(
        'ChdSZXF1ZXN0TG9naW5Db2RlUmVxdWVzdBIXCgd1c2VyX2lkGAEgASgJUgZ1c2VySWQ=');

@$core.Deprecated('Use requestLoginCodeResponseDescriptor instead')
const RequestLoginCodeResponse$json = {
  '1': 'RequestLoginCodeResponse',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 9, '10': 'code'},
    {'1': 'ttl_sec', '3': 2, '4': 1, '5': 3, '10': 'ttlSec'},
    {'1': 'address_hint', '3': 3, '4': 1, '5': 9, '10': 'addressHint'},
  ],
};

/// Descriptor for `RequestLoginCodeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestLoginCodeResponseDescriptor =
    $convert.base64Decode(
        'ChhSZXF1ZXN0TG9naW5Db2RlUmVzcG9uc2USEgoEY29kZRgBIAEoCVIEY29kZRIXCgd0dGxfc2'
        'VjGAIgASgDUgZ0dGxTZWMSIQoMYWRkcmVzc19oaW50GAMgASgJUgthZGRyZXNzSGludA==');

@$core.Deprecated('Use setPayoutAddressRequestDescriptor instead')
const SetPayoutAddressRequest$json = {
  '1': 'SetPayoutAddressRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
    {'1': 'address', '3': 2, '4': 1, '5': 9, '10': 'address'},
    {'1': 'signature', '3': 3, '4': 1, '5': 9, '10': 'signature'},
    {'1': 'code', '3': 4, '4': 1, '5': 9, '10': 'code'},
  ],
};

/// Descriptor for `SetPayoutAddressRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPayoutAddressRequestDescriptor = $convert.base64Decode(
    'ChdTZXRQYXlvdXRBZGRyZXNzUmVxdWVzdBIUCgV0b2tlbhgBIAEoCVIFdG9rZW4SGAoHYWRkcm'
    'VzcxgCIAEoCVIHYWRkcmVzcxIcCglzaWduYXR1cmUYAyABKAlSCXNpZ25hdHVyZRISCgRjb2Rl'
    'GAQgASgJUgRjb2Rl');

@$core.Deprecated('Use setPayoutAddressResponseDescriptor instead')
const SetPayoutAddressResponse$json = {
  '1': 'SetPayoutAddressResponse',
  '2': [
    {'1': 'ok', '3': 1, '4': 1, '5': 8, '10': 'ok'},
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
    {'1': 'address', '3': 3, '4': 1, '5': 9, '10': 'address'},
  ],
};

/// Descriptor for `SetPayoutAddressResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setPayoutAddressResponseDescriptor =
    $convert.base64Decode(
        'ChhTZXRQYXlvdXRBZGRyZXNzUmVzcG9uc2USDgoCb2sYASABKAhSAm9rEhQKBWVycm9yGAIgAS'
        'gJUgVlcnJvchIYCgdhZGRyZXNzGAMgASgJUgdhZGRyZXNz');

@$core.Deprecated('Use loginResponseDescriptor instead')
const LoginResponse$json = {
  '1': 'LoginResponse',
  '2': [
    {'1': 'ok', '3': 1, '4': 1, '5': 8, '10': 'ok'},
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
    {'1': 'token', '3': 3, '4': 1, '5': 9, '10': 'token'},
    {'1': 'user_id', '3': 4, '4': 1, '5': 9, '10': 'userId'},
    {'1': 'nickname', '3': 5, '4': 1, '5': 9, '10': 'nickname'},
    {'1': 'payout_address', '3': 6, '4': 1, '5': 9, '10': 'payoutAddress'},
  ],
};

/// Descriptor for `LoginResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List loginResponseDescriptor = $convert.base64Decode(
    'Cg1Mb2dpblJlc3BvbnNlEg4KAm9rGAEgASgIUgJvaxIUCgVlcnJvchgCIAEoCVIFZXJyb3ISFA'
    'oFdG9rZW4YAyABKAlSBXRva2VuEhcKB3VzZXJfaWQYBCABKAlSBnVzZXJJZBIaCghuaWNrbmFt'
    'ZRgFIAEoCVIIbmlja25hbWUSJQoOcGF5b3V0X2FkZHJlc3MYBiABKAlSDXBheW91dEFkZHJlc3'
    'M=');

@$core.Deprecated('Use logoutRequestDescriptor instead')
const LogoutRequest$json = {
  '1': 'LogoutRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
  ],
};

/// Descriptor for `LogoutRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logoutRequestDescriptor = $convert
    .base64Decode('Cg1Mb2dvdXRSZXF1ZXN0EhQKBXRva2VuGAEgASgJUgV0b2tlbg==');

@$core.Deprecated('Use logoutResponseDescriptor instead')
const LogoutResponse$json = {
  '1': 'LogoutResponse',
  '2': [
    {'1': 'ok', '3': 1, '4': 1, '5': 8, '10': 'ok'},
  ],
};

/// Descriptor for `LogoutResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logoutResponseDescriptor =
    $convert.base64Decode('Cg5Mb2dvdXRSZXNwb25zZRIOCgJvaxgBIAEoCFICb2s=');

@$core.Deprecated('Use getUserInfoRequestDescriptor instead')
const GetUserInfoRequest$json = {
  '1': 'GetUserInfoRequest',
  '2': [
    {'1': 'token', '3': 1, '4': 1, '5': 9, '10': 'token'},
  ],
};

/// Descriptor for `GetUserInfoRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserInfoRequestDescriptor = $convert
    .base64Decode('ChJHZXRVc2VySW5mb1JlcXVlc3QSFAoFdG9rZW4YASABKAlSBXRva2Vu');

@$core.Deprecated('Use getUserInfoResponseDescriptor instead')
const GetUserInfoResponse$json = {
  '1': 'GetUserInfoResponse',
  '2': [
    {'1': 'user_id', '3': 1, '4': 1, '5': 9, '10': 'userId'},
    {'1': 'nickname', '3': 2, '4': 1, '5': 9, '10': 'nickname'},
    {'1': 'created', '3': 3, '4': 1, '5': 3, '10': 'created'},
    {'1': 'last_login', '3': 4, '4': 1, '5': 3, '10': 'lastLogin'},
  ],
};

/// Descriptor for `GetUserInfoResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserInfoResponseDescriptor = $convert.base64Decode(
    'ChNHZXRVc2VySW5mb1Jlc3BvbnNlEhcKB3VzZXJfaWQYASABKAlSBnVzZXJJZBIaCghuaWNrbm'
    'FtZRgCIAEoCVIIbmlja25hbWUSGAoHY3JlYXRlZBgDIAEoA1IHY3JlYXRlZBIdCgpsYXN0X2xv'
    'Z2luGAQgASgDUglsYXN0TG9naW4=');
