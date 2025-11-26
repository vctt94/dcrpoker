// This is a generated file - do not edit.
//
// Generated from pokerreferee.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'pokerreferee.pb.dart' as $0;

export 'pokerreferee.pb.dart';

/// PokerReferee coordinates Schnorr adaptor escrow/presign/finalize for
/// SNG/WTA tables (2–6 seats, single winner).
@$pb.GrpcServiceName('poker.PokerReferee')
class PokerRefereeClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  PokerRefereeClient(super.channel, {super.options, super.interceptors});

  /// Creates a per-player escrow session for a table/session/seat.
  $grpc.ResponseFuture<$0.OpenEscrowResponse> openEscrow(
    $0.OpenEscrowRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$openEscrow, request, options: options);
  }

  /// Binds an existing escrow to a table/session seat and reports readiness.
  $grpc.ResponseFuture<$0.BindEscrowResponse> bindEscrow(
    $0.BindEscrowRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$bindEscrow, request, options: options);
  }

  /// Publishes the compressed session pubkey used for adaptor presign.
  $grpc.ResponseFuture<$0.PublishSessionKeyResponse> publishSessionKey(
    $0.PublishSessionKeyRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$publishSessionKey, request, options: options);
  }

  /// Bidirectional stream for presign handshakes (HELLO → REQ → presigs).
  $grpc.ResponseStream<$0.SettlementStreamMessage> settlementStream(
    $async.Stream<$0.SettlementStreamMessage> request, {
    $grpc.CallOptions? options,
  }) {
    return $createStreamingCall(_$settlementStream, request, options: options);
  }

  /// Returns the draft, gamma, and presigs for the winning branch.
  $grpc.ResponseFuture<$0.GetFinalizeBundleResponse> getFinalizeBundle(
    $0.GetFinalizeBundleRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getFinalizeBundle, request, options: options);
  }

  /// Returns funding/conf status for an escrow owned by the caller.
  $grpc.ResponseFuture<$0.GetEscrowStatusResponse> getEscrowStatus(
    $0.GetEscrowStatusRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$getEscrowStatus, request, options: options);
  }

  $grpc.ResponseFuture<$0.SetPayoutAddressResponse> setPayoutAddress(
    $0.SetPayoutAddressRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$setPayoutAddress, request, options: options);
  }

  // method descriptors

  static final _$openEscrow =
      $grpc.ClientMethod<$0.OpenEscrowRequest, $0.OpenEscrowResponse>(
          '/poker.PokerReferee/OpenEscrow',
          ($0.OpenEscrowRequest value) => value.writeToBuffer(),
          $0.OpenEscrowResponse.fromBuffer);
  static final _$bindEscrow =
      $grpc.ClientMethod<$0.BindEscrowRequest, $0.BindEscrowResponse>(
          '/poker.PokerReferee/BindEscrow',
          ($0.BindEscrowRequest value) => value.writeToBuffer(),
          $0.BindEscrowResponse.fromBuffer);
  static final _$publishSessionKey = $grpc.ClientMethod<
          $0.PublishSessionKeyRequest, $0.PublishSessionKeyResponse>(
      '/poker.PokerReferee/PublishSessionKey',
      ($0.PublishSessionKeyRequest value) => value.writeToBuffer(),
      $0.PublishSessionKeyResponse.fromBuffer);
  static final _$settlementStream = $grpc.ClientMethod<
          $0.SettlementStreamMessage, $0.SettlementStreamMessage>(
      '/poker.PokerReferee/SettlementStream',
      ($0.SettlementStreamMessage value) => value.writeToBuffer(),
      $0.SettlementStreamMessage.fromBuffer);
  static final _$getFinalizeBundle = $grpc.ClientMethod<
          $0.GetFinalizeBundleRequest, $0.GetFinalizeBundleResponse>(
      '/poker.PokerReferee/GetFinalizeBundle',
      ($0.GetFinalizeBundleRequest value) => value.writeToBuffer(),
      $0.GetFinalizeBundleResponse.fromBuffer);
  static final _$getEscrowStatus =
      $grpc.ClientMethod<$0.GetEscrowStatusRequest, $0.GetEscrowStatusResponse>(
          '/poker.PokerReferee/GetEscrowStatus',
          ($0.GetEscrowStatusRequest value) => value.writeToBuffer(),
          $0.GetEscrowStatusResponse.fromBuffer);
  static final _$setPayoutAddress = $grpc.ClientMethod<
          $0.SetPayoutAddressRequest, $0.SetPayoutAddressResponse>(
      '/poker.PokerReferee/SetPayoutAddress',
      ($0.SetPayoutAddressRequest value) => value.writeToBuffer(),
      $0.SetPayoutAddressResponse.fromBuffer);
}

@$pb.GrpcServiceName('poker.PokerReferee')
abstract class PokerRefereeServiceBase extends $grpc.Service {
  $core.String get $name => 'poker.PokerReferee';

  PokerRefereeServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.OpenEscrowRequest, $0.OpenEscrowResponse>(
        'OpenEscrow',
        openEscrow_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.OpenEscrowRequest.fromBuffer(value),
        ($0.OpenEscrowResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.BindEscrowRequest, $0.BindEscrowResponse>(
        'BindEscrow',
        bindEscrow_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.BindEscrowRequest.fromBuffer(value),
        ($0.BindEscrowResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.PublishSessionKeyRequest,
            $0.PublishSessionKeyResponse>(
        'PublishSessionKey',
        publishSessionKey_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.PublishSessionKeyRequest.fromBuffer(value),
        ($0.PublishSessionKeyResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SettlementStreamMessage,
            $0.SettlementStreamMessage>(
        'SettlementStream',
        settlementStream,
        true,
        true,
        ($core.List<$core.int> value) =>
            $0.SettlementStreamMessage.fromBuffer(value),
        ($0.SettlementStreamMessage value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetFinalizeBundleRequest,
            $0.GetFinalizeBundleResponse>(
        'GetFinalizeBundle',
        getFinalizeBundle_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.GetFinalizeBundleRequest.fromBuffer(value),
        ($0.GetFinalizeBundleResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetEscrowStatusRequest,
            $0.GetEscrowStatusResponse>(
        'GetEscrowStatus',
        getEscrowStatus_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.GetEscrowStatusRequest.fromBuffer(value),
        ($0.GetEscrowStatusResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SetPayoutAddressRequest,
            $0.SetPayoutAddressResponse>(
        'SetPayoutAddress',
        setPayoutAddress_Pre,
        false,
        false,
        ($core.List<$core.int> value) =>
            $0.SetPayoutAddressRequest.fromBuffer(value),
        ($0.SetPayoutAddressResponse value) => value.writeToBuffer()));
  }

  $async.Future<$0.OpenEscrowResponse> openEscrow_Pre($grpc.ServiceCall $call,
      $async.Future<$0.OpenEscrowRequest> $request) async {
    return openEscrow($call, await $request);
  }

  $async.Future<$0.OpenEscrowResponse> openEscrow(
      $grpc.ServiceCall call, $0.OpenEscrowRequest request);

  $async.Future<$0.BindEscrowResponse> bindEscrow_Pre($grpc.ServiceCall $call,
      $async.Future<$0.BindEscrowRequest> $request) async {
    return bindEscrow($call, await $request);
  }

  $async.Future<$0.BindEscrowResponse> bindEscrow(
      $grpc.ServiceCall call, $0.BindEscrowRequest request);

  $async.Future<$0.PublishSessionKeyResponse> publishSessionKey_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.PublishSessionKeyRequest> $request) async {
    return publishSessionKey($call, await $request);
  }

  $async.Future<$0.PublishSessionKeyResponse> publishSessionKey(
      $grpc.ServiceCall call, $0.PublishSessionKeyRequest request);

  $async.Stream<$0.SettlementStreamMessage> settlementStream(
      $grpc.ServiceCall call,
      $async.Stream<$0.SettlementStreamMessage> request);

  $async.Future<$0.GetFinalizeBundleResponse> getFinalizeBundle_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.GetFinalizeBundleRequest> $request) async {
    return getFinalizeBundle($call, await $request);
  }

  $async.Future<$0.GetFinalizeBundleResponse> getFinalizeBundle(
      $grpc.ServiceCall call, $0.GetFinalizeBundleRequest request);

  $async.Future<$0.GetEscrowStatusResponse> getEscrowStatus_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.GetEscrowStatusRequest> $request) async {
    return getEscrowStatus($call, await $request);
  }

  $async.Future<$0.GetEscrowStatusResponse> getEscrowStatus(
      $grpc.ServiceCall call, $0.GetEscrowStatusRequest request);

  $async.Future<$0.SetPayoutAddressResponse> setPayoutAddress_Pre(
      $grpc.ServiceCall $call,
      $async.Future<$0.SetPayoutAddressRequest> $request) async {
    return setPayoutAddress($call, await $request);
  }

  $async.Future<$0.SetPayoutAddressResponse> setPayoutAddress(
      $grpc.ServiceCall call, $0.SetPayoutAddressRequest request);
}
