//
//  Generated code. Do not modify.
//  source: poker.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'poker.pb.dart' as $0;

export 'poker.pb.dart';

@$pb.GrpcServiceName('poker.PokerService')
class PokerServiceClient extends $grpc.Client {
  static final _$startGameStream = $grpc.ClientMethod<$0.StartGameStreamRequest, $0.GameUpdate>(
      '/poker.PokerService/StartGameStream',
      ($0.StartGameStreamRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GameUpdate.fromBuffer(value));
  static final _$showCards = $grpc.ClientMethod<$0.ShowCardsRequest, $0.ShowCardsResponse>(
      '/poker.PokerService/ShowCards',
      ($0.ShowCardsRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.ShowCardsResponse.fromBuffer(value));
  static final _$hideCards = $grpc.ClientMethod<$0.HideCardsRequest, $0.HideCardsResponse>(
      '/poker.PokerService/HideCards',
      ($0.HideCardsRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.HideCardsResponse.fromBuffer(value));
  static final _$makeBet = $grpc.ClientMethod<$0.MakeBetRequest, $0.MakeBetResponse>(
      '/poker.PokerService/MakeBet',
      ($0.MakeBetRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.MakeBetResponse.fromBuffer(value));
  static final _$callBet = $grpc.ClientMethod<$0.CallBetRequest, $0.CallBetResponse>(
      '/poker.PokerService/CallBet',
      ($0.CallBetRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.CallBetResponse.fromBuffer(value));
  static final _$foldBet = $grpc.ClientMethod<$0.FoldBetRequest, $0.FoldBetResponse>(
      '/poker.PokerService/FoldBet',
      ($0.FoldBetRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.FoldBetResponse.fromBuffer(value));
  static final _$checkBet = $grpc.ClientMethod<$0.CheckBetRequest, $0.CheckBetResponse>(
      '/poker.PokerService/CheckBet',
      ($0.CheckBetRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.CheckBetResponse.fromBuffer(value));
  static final _$getGameState = $grpc.ClientMethod<$0.GetGameStateRequest, $0.GetGameStateResponse>(
      '/poker.PokerService/GetGameState',
      ($0.GetGameStateRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetGameStateResponse.fromBuffer(value));
  static final _$evaluateHand = $grpc.ClientMethod<$0.EvaluateHandRequest, $0.EvaluateHandResponse>(
      '/poker.PokerService/EvaluateHand',
      ($0.EvaluateHandRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.EvaluateHandResponse.fromBuffer(value));
  static final _$getLastWinners = $grpc.ClientMethod<$0.GetLastWinnersRequest, $0.GetLastWinnersResponse>(
      '/poker.PokerService/GetLastWinners',
      ($0.GetLastWinnersRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetLastWinnersResponse.fromBuffer(value));

  PokerServiceClient($grpc.ClientChannel channel,
      {$grpc.CallOptions? options,
      $core.Iterable<$grpc.ClientInterceptor>? interceptors})
      : super(channel, options: options,
        interceptors: interceptors);

  $grpc.ResponseStream<$0.GameUpdate> startGameStream($0.StartGameStreamRequest request, {$grpc.CallOptions? options}) {
    return $createStreamingCall(_$startGameStream, $async.Stream.fromIterable([request]), options: options);
  }

  $grpc.ResponseFuture<$0.ShowCardsResponse> showCards($0.ShowCardsRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$showCards, request, options: options);
  }

  $grpc.ResponseFuture<$0.HideCardsResponse> hideCards($0.HideCardsRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$hideCards, request, options: options);
  }

  $grpc.ResponseFuture<$0.MakeBetResponse> makeBet($0.MakeBetRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$makeBet, request, options: options);
  }

  $grpc.ResponseFuture<$0.CallBetResponse> callBet($0.CallBetRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$callBet, request, options: options);
  }

  $grpc.ResponseFuture<$0.FoldBetResponse> foldBet($0.FoldBetRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$foldBet, request, options: options);
  }

  $grpc.ResponseFuture<$0.CheckBetResponse> checkBet($0.CheckBetRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$checkBet, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetGameStateResponse> getGameState($0.GetGameStateRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getGameState, request, options: options);
  }

  $grpc.ResponseFuture<$0.EvaluateHandResponse> evaluateHand($0.EvaluateHandRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$evaluateHand, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetLastWinnersResponse> getLastWinners($0.GetLastWinnersRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getLastWinners, request, options: options);
  }
}

@$pb.GrpcServiceName('poker.PokerService')
abstract class PokerServiceBase extends $grpc.Service {
  $core.String get $name => 'poker.PokerService';

  PokerServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.StartGameStreamRequest, $0.GameUpdate>(
        'StartGameStream',
        startGameStream_Pre,
        false,
        true,
        ($core.List<$core.int> value) => $0.StartGameStreamRequest.fromBuffer(value),
        ($0.GameUpdate value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ShowCardsRequest, $0.ShowCardsResponse>(
        'ShowCards',
        showCards_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.ShowCardsRequest.fromBuffer(value),
        ($0.ShowCardsResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.HideCardsRequest, $0.HideCardsResponse>(
        'HideCards',
        hideCards_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.HideCardsRequest.fromBuffer(value),
        ($0.HideCardsResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.MakeBetRequest, $0.MakeBetResponse>(
        'MakeBet',
        makeBet_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.MakeBetRequest.fromBuffer(value),
        ($0.MakeBetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.CallBetRequest, $0.CallBetResponse>(
        'CallBet',
        callBet_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.CallBetRequest.fromBuffer(value),
        ($0.CallBetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.FoldBetRequest, $0.FoldBetResponse>(
        'FoldBet',
        foldBet_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.FoldBetRequest.fromBuffer(value),
        ($0.FoldBetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.CheckBetRequest, $0.CheckBetResponse>(
        'CheckBet',
        checkBet_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.CheckBetRequest.fromBuffer(value),
        ($0.CheckBetResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetGameStateRequest, $0.GetGameStateResponse>(
        'GetGameState',
        getGameState_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetGameStateRequest.fromBuffer(value),
        ($0.GetGameStateResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.EvaluateHandRequest, $0.EvaluateHandResponse>(
        'EvaluateHand',
        evaluateHand_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.EvaluateHandRequest.fromBuffer(value),
        ($0.EvaluateHandResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetLastWinnersRequest, $0.GetLastWinnersResponse>(
        'GetLastWinners',
        getLastWinners_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetLastWinnersRequest.fromBuffer(value),
        ($0.GetLastWinnersResponse value) => value.writeToBuffer()));
  }

  $async.Stream<$0.GameUpdate> startGameStream_Pre($grpc.ServiceCall call, $async.Future<$0.StartGameStreamRequest> request) async* {
    yield* startGameStream(call, await request);
  }

  $async.Future<$0.ShowCardsResponse> showCards_Pre($grpc.ServiceCall call, $async.Future<$0.ShowCardsRequest> request) async {
    return showCards(call, await request);
  }

  $async.Future<$0.HideCardsResponse> hideCards_Pre($grpc.ServiceCall call, $async.Future<$0.HideCardsRequest> request) async {
    return hideCards(call, await request);
  }

  $async.Future<$0.MakeBetResponse> makeBet_Pre($grpc.ServiceCall call, $async.Future<$0.MakeBetRequest> request) async {
    return makeBet(call, await request);
  }

  $async.Future<$0.CallBetResponse> callBet_Pre($grpc.ServiceCall call, $async.Future<$0.CallBetRequest> request) async {
    return callBet(call, await request);
  }

  $async.Future<$0.FoldBetResponse> foldBet_Pre($grpc.ServiceCall call, $async.Future<$0.FoldBetRequest> request) async {
    return foldBet(call, await request);
  }

  $async.Future<$0.CheckBetResponse> checkBet_Pre($grpc.ServiceCall call, $async.Future<$0.CheckBetRequest> request) async {
    return checkBet(call, await request);
  }

  $async.Future<$0.GetGameStateResponse> getGameState_Pre($grpc.ServiceCall call, $async.Future<$0.GetGameStateRequest> request) async {
    return getGameState(call, await request);
  }

  $async.Future<$0.EvaluateHandResponse> evaluateHand_Pre($grpc.ServiceCall call, $async.Future<$0.EvaluateHandRequest> request) async {
    return evaluateHand(call, await request);
  }

  $async.Future<$0.GetLastWinnersResponse> getLastWinners_Pre($grpc.ServiceCall call, $async.Future<$0.GetLastWinnersRequest> request) async {
    return getLastWinners(call, await request);
  }

  $async.Stream<$0.GameUpdate> startGameStream($grpc.ServiceCall call, $0.StartGameStreamRequest request);
  $async.Future<$0.ShowCardsResponse> showCards($grpc.ServiceCall call, $0.ShowCardsRequest request);
  $async.Future<$0.HideCardsResponse> hideCards($grpc.ServiceCall call, $0.HideCardsRequest request);
  $async.Future<$0.MakeBetResponse> makeBet($grpc.ServiceCall call, $0.MakeBetRequest request);
  $async.Future<$0.CallBetResponse> callBet($grpc.ServiceCall call, $0.CallBetRequest request);
  $async.Future<$0.FoldBetResponse> foldBet($grpc.ServiceCall call, $0.FoldBetRequest request);
  $async.Future<$0.CheckBetResponse> checkBet($grpc.ServiceCall call, $0.CheckBetRequest request);
  $async.Future<$0.GetGameStateResponse> getGameState($grpc.ServiceCall call, $0.GetGameStateRequest request);
  $async.Future<$0.EvaluateHandResponse> evaluateHand($grpc.ServiceCall call, $0.EvaluateHandRequest request);
  $async.Future<$0.GetLastWinnersResponse> getLastWinners($grpc.ServiceCall call, $0.GetLastWinnersRequest request);
}
@$pb.GrpcServiceName('poker.LobbyService')
class LobbyServiceClient extends $grpc.Client {
  static final _$createTable = $grpc.ClientMethod<$0.CreateTableRequest, $0.CreateTableResponse>(
      '/poker.LobbyService/CreateTable',
      ($0.CreateTableRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.CreateTableResponse.fromBuffer(value));
  static final _$joinTable = $grpc.ClientMethod<$0.JoinTableRequest, $0.JoinTableResponse>(
      '/poker.LobbyService/JoinTable',
      ($0.JoinTableRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.JoinTableResponse.fromBuffer(value));
  static final _$leaveTable = $grpc.ClientMethod<$0.LeaveTableRequest, $0.LeaveTableResponse>(
      '/poker.LobbyService/LeaveTable',
      ($0.LeaveTableRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.LeaveTableResponse.fromBuffer(value));
  static final _$getTables = $grpc.ClientMethod<$0.GetTablesRequest, $0.GetTablesResponse>(
      '/poker.LobbyService/GetTables',
      ($0.GetTablesRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetTablesResponse.fromBuffer(value));
  static final _$getPlayerCurrentTable = $grpc.ClientMethod<$0.GetPlayerCurrentTableRequest, $0.GetPlayerCurrentTableResponse>(
      '/poker.LobbyService/GetPlayerCurrentTable',
      ($0.GetPlayerCurrentTableRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetPlayerCurrentTableResponse.fromBuffer(value));
  static final _$getBalance = $grpc.ClientMethod<$0.GetBalanceRequest, $0.GetBalanceResponse>(
      '/poker.LobbyService/GetBalance',
      ($0.GetBalanceRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetBalanceResponse.fromBuffer(value));
  static final _$updateBalance = $grpc.ClientMethod<$0.UpdateBalanceRequest, $0.UpdateBalanceResponse>(
      '/poker.LobbyService/UpdateBalance',
      ($0.UpdateBalanceRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.UpdateBalanceResponse.fromBuffer(value));
  static final _$processTip = $grpc.ClientMethod<$0.ProcessTipRequest, $0.ProcessTipResponse>(
      '/poker.LobbyService/ProcessTip',
      ($0.ProcessTipRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.ProcessTipResponse.fromBuffer(value));
  static final _$setPlayerReady = $grpc.ClientMethod<$0.SetPlayerReadyRequest, $0.SetPlayerReadyResponse>(
      '/poker.LobbyService/SetPlayerReady',
      ($0.SetPlayerReadyRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.SetPlayerReadyResponse.fromBuffer(value));
  static final _$setPlayerUnready = $grpc.ClientMethod<$0.SetPlayerUnreadyRequest, $0.SetPlayerUnreadyResponse>(
      '/poker.LobbyService/SetPlayerUnready',
      ($0.SetPlayerUnreadyRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.SetPlayerUnreadyResponse.fromBuffer(value));
  static final _$startNotificationStream = $grpc.ClientMethod<$0.StartNotificationStreamRequest, $0.Notification>(
      '/poker.LobbyService/StartNotificationStream',
      ($0.StartNotificationStreamRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.Notification.fromBuffer(value));

  LobbyServiceClient($grpc.ClientChannel channel,
      {$grpc.CallOptions? options,
      $core.Iterable<$grpc.ClientInterceptor>? interceptors})
      : super(channel, options: options,
        interceptors: interceptors);

  $grpc.ResponseFuture<$0.CreateTableResponse> createTable($0.CreateTableRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$createTable, request, options: options);
  }

  $grpc.ResponseFuture<$0.JoinTableResponse> joinTable($0.JoinTableRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$joinTable, request, options: options);
  }

  $grpc.ResponseFuture<$0.LeaveTableResponse> leaveTable($0.LeaveTableRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$leaveTable, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetTablesResponse> getTables($0.GetTablesRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getTables, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetPlayerCurrentTableResponse> getPlayerCurrentTable($0.GetPlayerCurrentTableRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getPlayerCurrentTable, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetBalanceResponse> getBalance($0.GetBalanceRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getBalance, request, options: options);
  }

  $grpc.ResponseFuture<$0.UpdateBalanceResponse> updateBalance($0.UpdateBalanceRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$updateBalance, request, options: options);
  }

  $grpc.ResponseFuture<$0.ProcessTipResponse> processTip($0.ProcessTipRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$processTip, request, options: options);
  }

  $grpc.ResponseFuture<$0.SetPlayerReadyResponse> setPlayerReady($0.SetPlayerReadyRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$setPlayerReady, request, options: options);
  }

  $grpc.ResponseFuture<$0.SetPlayerUnreadyResponse> setPlayerUnready($0.SetPlayerUnreadyRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$setPlayerUnready, request, options: options);
  }

  $grpc.ResponseStream<$0.Notification> startNotificationStream($0.StartNotificationStreamRequest request, {$grpc.CallOptions? options}) {
    return $createStreamingCall(_$startNotificationStream, $async.Stream.fromIterable([request]), options: options);
  }
}

@$pb.GrpcServiceName('poker.LobbyService')
abstract class LobbyServiceBase extends $grpc.Service {
  $core.String get $name => 'poker.LobbyService';

  LobbyServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.CreateTableRequest, $0.CreateTableResponse>(
        'CreateTable',
        createTable_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.CreateTableRequest.fromBuffer(value),
        ($0.CreateTableResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.JoinTableRequest, $0.JoinTableResponse>(
        'JoinTable',
        joinTable_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.JoinTableRequest.fromBuffer(value),
        ($0.JoinTableResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.LeaveTableRequest, $0.LeaveTableResponse>(
        'LeaveTable',
        leaveTable_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.LeaveTableRequest.fromBuffer(value),
        ($0.LeaveTableResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetTablesRequest, $0.GetTablesResponse>(
        'GetTables',
        getTables_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetTablesRequest.fromBuffer(value),
        ($0.GetTablesResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetPlayerCurrentTableRequest, $0.GetPlayerCurrentTableResponse>(
        'GetPlayerCurrentTable',
        getPlayerCurrentTable_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetPlayerCurrentTableRequest.fromBuffer(value),
        ($0.GetPlayerCurrentTableResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetBalanceRequest, $0.GetBalanceResponse>(
        'GetBalance',
        getBalance_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetBalanceRequest.fromBuffer(value),
        ($0.GetBalanceResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.UpdateBalanceRequest, $0.UpdateBalanceResponse>(
        'UpdateBalance',
        updateBalance_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.UpdateBalanceRequest.fromBuffer(value),
        ($0.UpdateBalanceResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.ProcessTipRequest, $0.ProcessTipResponse>(
        'ProcessTip',
        processTip_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.ProcessTipRequest.fromBuffer(value),
        ($0.ProcessTipResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SetPlayerReadyRequest, $0.SetPlayerReadyResponse>(
        'SetPlayerReady',
        setPlayerReady_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SetPlayerReadyRequest.fromBuffer(value),
        ($0.SetPlayerReadyResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SetPlayerUnreadyRequest, $0.SetPlayerUnreadyResponse>(
        'SetPlayerUnready',
        setPlayerUnready_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SetPlayerUnreadyRequest.fromBuffer(value),
        ($0.SetPlayerUnreadyResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.StartNotificationStreamRequest, $0.Notification>(
        'StartNotificationStream',
        startNotificationStream_Pre,
        false,
        true,
        ($core.List<$core.int> value) => $0.StartNotificationStreamRequest.fromBuffer(value),
        ($0.Notification value) => value.writeToBuffer()));
  }

  $async.Future<$0.CreateTableResponse> createTable_Pre($grpc.ServiceCall call, $async.Future<$0.CreateTableRequest> request) async {
    return createTable(call, await request);
  }

  $async.Future<$0.JoinTableResponse> joinTable_Pre($grpc.ServiceCall call, $async.Future<$0.JoinTableRequest> request) async {
    return joinTable(call, await request);
  }

  $async.Future<$0.LeaveTableResponse> leaveTable_Pre($grpc.ServiceCall call, $async.Future<$0.LeaveTableRequest> request) async {
    return leaveTable(call, await request);
  }

  $async.Future<$0.GetTablesResponse> getTables_Pre($grpc.ServiceCall call, $async.Future<$0.GetTablesRequest> request) async {
    return getTables(call, await request);
  }

  $async.Future<$0.GetPlayerCurrentTableResponse> getPlayerCurrentTable_Pre($grpc.ServiceCall call, $async.Future<$0.GetPlayerCurrentTableRequest> request) async {
    return getPlayerCurrentTable(call, await request);
  }

  $async.Future<$0.GetBalanceResponse> getBalance_Pre($grpc.ServiceCall call, $async.Future<$0.GetBalanceRequest> request) async {
    return getBalance(call, await request);
  }

  $async.Future<$0.UpdateBalanceResponse> updateBalance_Pre($grpc.ServiceCall call, $async.Future<$0.UpdateBalanceRequest> request) async {
    return updateBalance(call, await request);
  }

  $async.Future<$0.ProcessTipResponse> processTip_Pre($grpc.ServiceCall call, $async.Future<$0.ProcessTipRequest> request) async {
    return processTip(call, await request);
  }

  $async.Future<$0.SetPlayerReadyResponse> setPlayerReady_Pre($grpc.ServiceCall call, $async.Future<$0.SetPlayerReadyRequest> request) async {
    return setPlayerReady(call, await request);
  }

  $async.Future<$0.SetPlayerUnreadyResponse> setPlayerUnready_Pre($grpc.ServiceCall call, $async.Future<$0.SetPlayerUnreadyRequest> request) async {
    return setPlayerUnready(call, await request);
  }

  $async.Stream<$0.Notification> startNotificationStream_Pre($grpc.ServiceCall call, $async.Future<$0.StartNotificationStreamRequest> request) async* {
    yield* startNotificationStream(call, await request);
  }

  $async.Future<$0.CreateTableResponse> createTable($grpc.ServiceCall call, $0.CreateTableRequest request);
  $async.Future<$0.JoinTableResponse> joinTable($grpc.ServiceCall call, $0.JoinTableRequest request);
  $async.Future<$0.LeaveTableResponse> leaveTable($grpc.ServiceCall call, $0.LeaveTableRequest request);
  $async.Future<$0.GetTablesResponse> getTables($grpc.ServiceCall call, $0.GetTablesRequest request);
  $async.Future<$0.GetPlayerCurrentTableResponse> getPlayerCurrentTable($grpc.ServiceCall call, $0.GetPlayerCurrentTableRequest request);
  $async.Future<$0.GetBalanceResponse> getBalance($grpc.ServiceCall call, $0.GetBalanceRequest request);
  $async.Future<$0.UpdateBalanceResponse> updateBalance($grpc.ServiceCall call, $0.UpdateBalanceRequest request);
  $async.Future<$0.ProcessTipResponse> processTip($grpc.ServiceCall call, $0.ProcessTipRequest request);
  $async.Future<$0.SetPlayerReadyResponse> setPlayerReady($grpc.ServiceCall call, $0.SetPlayerReadyRequest request);
  $async.Future<$0.SetPlayerUnreadyResponse> setPlayerUnready($grpc.ServiceCall call, $0.SetPlayerUnreadyRequest request);
  $async.Stream<$0.Notification> startNotificationStream($grpc.ServiceCall call, $0.StartNotificationStreamRequest request);
}
