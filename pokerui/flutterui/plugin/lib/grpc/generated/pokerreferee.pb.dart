// This is a generated file - do not edit.
//
// Generated from pokerreferee.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class OpenEscrowRequest extends $pb.GeneratedMessage {
  factory OpenEscrowRequest({
    $fixnum.Int64? amountAtoms,
    $core.int? csvBlocks,
    $core.String? token,
    $core.List<$core.int>? compPubkey,
  }) {
    final result = create();
    if (amountAtoms != null) result.amountAtoms = amountAtoms;
    if (csvBlocks != null) result.csvBlocks = csvBlocks;
    if (token != null) result.token = token;
    if (compPubkey != null) result.compPubkey = compPubkey;
    return result;
  }

  OpenEscrowRequest._();

  factory OpenEscrowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenEscrowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenEscrowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'amountAtoms', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'csvBlocks', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'token')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'compPubkey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenEscrowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenEscrowRequest copyWith(void Function(OpenEscrowRequest) updates) =>
      super.copyWith((message) => updates(message as OpenEscrowRequest))
          as OpenEscrowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenEscrowRequest create() => OpenEscrowRequest._();
  @$core.override
  OpenEscrowRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenEscrowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenEscrowRequest>(create);
  static OpenEscrowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get amountAtoms => $_getI64(0);
  @$pb.TagNumber(1)
  set amountAtoms($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAmountAtoms() => $_has(0);
  @$pb.TagNumber(1)
  void clearAmountAtoms() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get csvBlocks => $_getIZ(1);
  @$pb.TagNumber(2)
  set csvBlocks($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCsvBlocks() => $_has(1);
  @$pb.TagNumber(2)
  void clearCsvBlocks() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get token => $_getSZ(2);
  @$pb.TagNumber(3)
  set token($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasToken() => $_has(2);
  @$pb.TagNumber(3)
  void clearToken() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get compPubkey => $_getN(3);
  @$pb.TagNumber(4)
  set compPubkey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCompPubkey() => $_has(3);
  @$pb.TagNumber(4)
  void clearCompPubkey() => $_clearField(4);
}

class OpenEscrowResponse extends $pb.GeneratedMessage {
  factory OpenEscrowResponse({
    $core.String? escrowId,
    $core.String? depositAddr,
    $core.String? redeemScriptHex,
    $core.String? pkScriptHex,
    $core.String? matchId,
    $core.int? requiredConfirmations,
  }) {
    final result = create();
    if (escrowId != null) result.escrowId = escrowId;
    if (depositAddr != null) result.depositAddr = depositAddr;
    if (redeemScriptHex != null) result.redeemScriptHex = redeemScriptHex;
    if (pkScriptHex != null) result.pkScriptHex = pkScriptHex;
    if (matchId != null) result.matchId = matchId;
    if (requiredConfirmations != null)
      result.requiredConfirmations = requiredConfirmations;
    return result;
  }

  OpenEscrowResponse._();

  factory OpenEscrowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenEscrowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenEscrowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'escrowId')
    ..aOS(2, _omitFieldNames ? '' : 'depositAddr')
    ..aOS(3, _omitFieldNames ? '' : 'redeemScriptHex')
    ..aOS(4, _omitFieldNames ? '' : 'pkScriptHex')
    ..aOS(5, _omitFieldNames ? '' : 'matchId')
    ..aI(6, _omitFieldNames ? '' : 'requiredConfirmations',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenEscrowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenEscrowResponse copyWith(void Function(OpenEscrowResponse) updates) =>
      super.copyWith((message) => updates(message as OpenEscrowResponse))
          as OpenEscrowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenEscrowResponse create() => OpenEscrowResponse._();
  @$core.override
  OpenEscrowResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenEscrowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenEscrowResponse>(create);
  static OpenEscrowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get escrowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set escrowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEscrowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEscrowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get depositAddr => $_getSZ(1);
  @$pb.TagNumber(2)
  set depositAddr($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDepositAddr() => $_has(1);
  @$pb.TagNumber(2)
  void clearDepositAddr() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get redeemScriptHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set redeemScriptHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRedeemScriptHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearRedeemScriptHex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get pkScriptHex => $_getSZ(3);
  @$pb.TagNumber(4)
  set pkScriptHex($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPkScriptHex() => $_has(3);
  @$pb.TagNumber(4)
  void clearPkScriptHex() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get matchId => $_getSZ(4);
  @$pb.TagNumber(5)
  set matchId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMatchId() => $_has(4);
  @$pb.TagNumber(5)
  void clearMatchId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get requiredConfirmations => $_getIZ(5);
  @$pb.TagNumber(6)
  set requiredConfirmations($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRequiredConfirmations() => $_has(5);
  @$pb.TagNumber(6)
  void clearRequiredConfirmations() => $_clearField(6);
}

class BindEscrowRequest extends $pb.GeneratedMessage {
  factory BindEscrowRequest({
    $core.String? outpoint,
    $core.String? tableId,
    $core.String? sessionId,
    $core.int? seatIndex,
    $core.String? matchId,
    $core.String? token,
    $core.String? redeemScriptHex,
    $core.int? csvBlocks,
  }) {
    final result = create();
    if (outpoint != null) result.outpoint = outpoint;
    if (tableId != null) result.tableId = tableId;
    if (sessionId != null) result.sessionId = sessionId;
    if (seatIndex != null) result.seatIndex = seatIndex;
    if (matchId != null) result.matchId = matchId;
    if (token != null) result.token = token;
    if (redeemScriptHex != null) result.redeemScriptHex = redeemScriptHex;
    if (csvBlocks != null) result.csvBlocks = csvBlocks;
    return result;
  }

  BindEscrowRequest._();

  factory BindEscrowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BindEscrowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BindEscrowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'outpoint')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..aOS(3, _omitFieldNames ? '' : 'sessionId')
    ..aI(4, _omitFieldNames ? '' : 'seatIndex', fieldType: $pb.PbFieldType.OU3)
    ..aOS(5, _omitFieldNames ? '' : 'matchId')
    ..aOS(6, _omitFieldNames ? '' : 'token')
    ..aOS(7, _omitFieldNames ? '' : 'redeemScriptHex')
    ..aI(8, _omitFieldNames ? '' : 'csvBlocks', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BindEscrowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BindEscrowRequest copyWith(void Function(BindEscrowRequest) updates) =>
      super.copyWith((message) => updates(message as BindEscrowRequest))
          as BindEscrowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BindEscrowRequest create() => BindEscrowRequest._();
  @$core.override
  BindEscrowRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BindEscrowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BindEscrowRequest>(create);
  static BindEscrowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get outpoint => $_getSZ(0);
  @$pb.TagNumber(1)
  set outpoint($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOutpoint() => $_has(0);
  @$pb.TagNumber(1)
  void clearOutpoint() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set sessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get seatIndex => $_getIZ(3);
  @$pb.TagNumber(4)
  set seatIndex($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSeatIndex() => $_has(3);
  @$pb.TagNumber(4)
  void clearSeatIndex() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get matchId => $_getSZ(4);
  @$pb.TagNumber(5)
  set matchId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMatchId() => $_has(4);
  @$pb.TagNumber(5)
  void clearMatchId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get token => $_getSZ(5);
  @$pb.TagNumber(6)
  set token($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasToken() => $_has(5);
  @$pb.TagNumber(6)
  void clearToken() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get redeemScriptHex => $_getSZ(6);
  @$pb.TagNumber(7)
  set redeemScriptHex($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRedeemScriptHex() => $_has(6);
  @$pb.TagNumber(7)
  void clearRedeemScriptHex() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get csvBlocks => $_getIZ(7);
  @$pb.TagNumber(8)
  set csvBlocks($core.int value) => $_setUnsignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCsvBlocks() => $_has(7);
  @$pb.TagNumber(8)
  void clearCsvBlocks() => $_clearField(8);
}

class BindEscrowResponse extends $pb.GeneratedMessage {
  factory BindEscrowResponse({
    $core.String? matchId,
    $core.String? tableId,
    $core.String? sessionId,
    $core.int? seatIndex,
    $core.String? escrowId,
    $core.bool? escrowReady,
    $fixnum.Int64? amountAtoms,
    $fixnum.Int64? requiredAmountAtoms,
    $core.String? outpoint,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (tableId != null) result.tableId = tableId;
    if (sessionId != null) result.sessionId = sessionId;
    if (seatIndex != null) result.seatIndex = seatIndex;
    if (escrowId != null) result.escrowId = escrowId;
    if (escrowReady != null) result.escrowReady = escrowReady;
    if (amountAtoms != null) result.amountAtoms = amountAtoms;
    if (requiredAmountAtoms != null)
      result.requiredAmountAtoms = requiredAmountAtoms;
    if (outpoint != null) result.outpoint = outpoint;
    return result;
  }

  BindEscrowResponse._();

  factory BindEscrowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BindEscrowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BindEscrowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aOS(2, _omitFieldNames ? '' : 'tableId')
    ..aOS(3, _omitFieldNames ? '' : 'sessionId')
    ..aI(4, _omitFieldNames ? '' : 'seatIndex', fieldType: $pb.PbFieldType.OU3)
    ..aOS(5, _omitFieldNames ? '' : 'escrowId')
    ..aOB(6, _omitFieldNames ? '' : 'escrowReady')
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'amountAtoms', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'requiredAmountAtoms', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(9, _omitFieldNames ? '' : 'outpoint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BindEscrowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BindEscrowResponse copyWith(void Function(BindEscrowResponse) updates) =>
      super.copyWith((message) => updates(message as BindEscrowResponse))
          as BindEscrowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BindEscrowResponse create() => BindEscrowResponse._();
  @$core.override
  BindEscrowResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BindEscrowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BindEscrowResponse>(create);
  static BindEscrowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tableId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tableId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTableId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTableId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set sessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get seatIndex => $_getIZ(3);
  @$pb.TagNumber(4)
  set seatIndex($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSeatIndex() => $_has(3);
  @$pb.TagNumber(4)
  void clearSeatIndex() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get escrowId => $_getSZ(4);
  @$pb.TagNumber(5)
  set escrowId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasEscrowId() => $_has(4);
  @$pb.TagNumber(5)
  void clearEscrowId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get escrowReady => $_getBF(5);
  @$pb.TagNumber(6)
  set escrowReady($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasEscrowReady() => $_has(5);
  @$pb.TagNumber(6)
  void clearEscrowReady() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get amountAtoms => $_getI64(6);
  @$pb.TagNumber(7)
  set amountAtoms($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAmountAtoms() => $_has(6);
  @$pb.TagNumber(7)
  void clearAmountAtoms() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get requiredAmountAtoms => $_getI64(7);
  @$pb.TagNumber(8)
  set requiredAmountAtoms($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRequiredAmountAtoms() => $_has(7);
  @$pb.TagNumber(8)
  void clearRequiredAmountAtoms() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get outpoint => $_getSZ(8);
  @$pb.TagNumber(9)
  set outpoint($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasOutpoint() => $_has(8);
  @$pb.TagNumber(9)
  void clearOutpoint() => $_clearField(9);
}

class GetEscrowStatusRequest extends $pb.GeneratedMessage {
  factory GetEscrowStatusRequest({
    $core.String? escrowId,
    $core.String? token,
  }) {
    final result = create();
    if (escrowId != null) result.escrowId = escrowId;
    if (token != null) result.token = token;
    return result;
  }

  GetEscrowStatusRequest._();

  factory GetEscrowStatusRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetEscrowStatusRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetEscrowStatusRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'escrowId')
    ..aOS(2, _omitFieldNames ? '' : 'token')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetEscrowStatusRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetEscrowStatusRequest copyWith(
          void Function(GetEscrowStatusRequest) updates) =>
      super.copyWith((message) => updates(message as GetEscrowStatusRequest))
          as GetEscrowStatusRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetEscrowStatusRequest create() => GetEscrowStatusRequest._();
  @$core.override
  GetEscrowStatusRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetEscrowStatusRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetEscrowStatusRequest>(create);
  static GetEscrowStatusRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get escrowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set escrowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEscrowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEscrowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get token => $_getSZ(1);
  @$pb.TagNumber(2)
  set token($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearToken() => $_clearField(2);
}

class GetEscrowStatusResponse extends $pb.GeneratedMessage {
  factory GetEscrowStatusResponse({
    $core.String? escrowId,
    $core.int? confs,
    $core.int? utxoCount,
    $core.bool? ok,
    $fixnum.Int64? updatedAtUnix,
    $core.String? fundingTxid,
    $core.int? fundingVout,
    $fixnum.Int64? amountAtoms,
    $core.String? pkScriptHex,
    $core.int? csvBlocks,
    $core.int? requiredConfirmations,
    $core.bool? matureForCsv,
  }) {
    final result = create();
    if (escrowId != null) result.escrowId = escrowId;
    if (confs != null) result.confs = confs;
    if (utxoCount != null) result.utxoCount = utxoCount;
    if (ok != null) result.ok = ok;
    if (updatedAtUnix != null) result.updatedAtUnix = updatedAtUnix;
    if (fundingTxid != null) result.fundingTxid = fundingTxid;
    if (fundingVout != null) result.fundingVout = fundingVout;
    if (amountAtoms != null) result.amountAtoms = amountAtoms;
    if (pkScriptHex != null) result.pkScriptHex = pkScriptHex;
    if (csvBlocks != null) result.csvBlocks = csvBlocks;
    if (requiredConfirmations != null)
      result.requiredConfirmations = requiredConfirmations;
    if (matureForCsv != null) result.matureForCsv = matureForCsv;
    return result;
  }

  GetEscrowStatusResponse._();

  factory GetEscrowStatusResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetEscrowStatusResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetEscrowStatusResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'escrowId')
    ..aI(2, _omitFieldNames ? '' : 'confs', fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'utxoCount', fieldType: $pb.PbFieldType.OU3)
    ..aOB(4, _omitFieldNames ? '' : 'ok')
    ..aInt64(5, _omitFieldNames ? '' : 'updatedAtUnix')
    ..aOS(6, _omitFieldNames ? '' : 'fundingTxid')
    ..aI(7, _omitFieldNames ? '' : 'fundingVout',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'amountAtoms', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(9, _omitFieldNames ? '' : 'pkScriptHex')
    ..aI(10, _omitFieldNames ? '' : 'csvBlocks', fieldType: $pb.PbFieldType.OU3)
    ..aI(11, _omitFieldNames ? '' : 'requiredConfirmations',
        fieldType: $pb.PbFieldType.OU3)
    ..aOB(12, _omitFieldNames ? '' : 'matureForCsv')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetEscrowStatusResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetEscrowStatusResponse copyWith(
          void Function(GetEscrowStatusResponse) updates) =>
      super.copyWith((message) => updates(message as GetEscrowStatusResponse))
          as GetEscrowStatusResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetEscrowStatusResponse create() => GetEscrowStatusResponse._();
  @$core.override
  GetEscrowStatusResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetEscrowStatusResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetEscrowStatusResponse>(create);
  static GetEscrowStatusResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get escrowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set escrowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEscrowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEscrowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get confs => $_getIZ(1);
  @$pb.TagNumber(2)
  set confs($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConfs() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfs() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get utxoCount => $_getIZ(2);
  @$pb.TagNumber(3)
  set utxoCount($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUtxoCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearUtxoCount() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get ok => $_getBF(3);
  @$pb.TagNumber(4)
  set ok($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOk() => $_has(3);
  @$pb.TagNumber(4)
  void clearOk() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get updatedAtUnix => $_getI64(4);
  @$pb.TagNumber(5)
  set updatedAtUnix($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasUpdatedAtUnix() => $_has(4);
  @$pb.TagNumber(5)
  void clearUpdatedAtUnix() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get fundingTxid => $_getSZ(5);
  @$pb.TagNumber(6)
  set fundingTxid($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFundingTxid() => $_has(5);
  @$pb.TagNumber(6)
  void clearFundingTxid() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get fundingVout => $_getIZ(6);
  @$pb.TagNumber(7)
  set fundingVout($core.int value) => $_setUnsignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasFundingVout() => $_has(6);
  @$pb.TagNumber(7)
  void clearFundingVout() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get amountAtoms => $_getI64(7);
  @$pb.TagNumber(8)
  set amountAtoms($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasAmountAtoms() => $_has(7);
  @$pb.TagNumber(8)
  void clearAmountAtoms() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get pkScriptHex => $_getSZ(8);
  @$pb.TagNumber(9)
  set pkScriptHex($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPkScriptHex() => $_has(8);
  @$pb.TagNumber(9)
  void clearPkScriptHex() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get csvBlocks => $_getIZ(9);
  @$pb.TagNumber(10)
  set csvBlocks($core.int value) => $_setUnsignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasCsvBlocks() => $_has(9);
  @$pb.TagNumber(10)
  void clearCsvBlocks() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get requiredConfirmations => $_getIZ(10);
  @$pb.TagNumber(11)
  set requiredConfirmations($core.int value) => $_setUnsignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasRequiredConfirmations() => $_has(10);
  @$pb.TagNumber(11)
  void clearRequiredConfirmations() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get matureForCsv => $_getBF(11);
  @$pb.TagNumber(12)
  set matureForCsv($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasMatureForCsv() => $_has(11);
  @$pb.TagNumber(12)
  void clearMatureForCsv() => $_clearField(12);
}

class PublishSessionKeyRequest extends $pb.GeneratedMessage {
  factory PublishSessionKeyRequest({
    $core.String? escrowId,
    $core.List<$core.int>? compPubkey,
  }) {
    final result = create();
    if (escrowId != null) result.escrowId = escrowId;
    if (compPubkey != null) result.compPubkey = compPubkey;
    return result;
  }

  PublishSessionKeyRequest._();

  factory PublishSessionKeyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PublishSessionKeyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PublishSessionKeyRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'escrowId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'compPubkey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishSessionKeyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishSessionKeyRequest copyWith(
          void Function(PublishSessionKeyRequest) updates) =>
      super.copyWith((message) => updates(message as PublishSessionKeyRequest))
          as PublishSessionKeyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PublishSessionKeyRequest create() => PublishSessionKeyRequest._();
  @$core.override
  PublishSessionKeyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PublishSessionKeyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PublishSessionKeyRequest>(create);
  static PublishSessionKeyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get escrowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set escrowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEscrowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEscrowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get compPubkey => $_getN(1);
  @$pb.TagNumber(2)
  set compPubkey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCompPubkey() => $_has(1);
  @$pb.TagNumber(2)
  void clearCompPubkey() => $_clearField(2);
}

class PublishSessionKeyResponse extends $pb.GeneratedMessage {
  factory PublishSessionKeyResponse({
    $core.bool? ok,
    $core.String? error,
  }) {
    final result = create();
    if (ok != null) result.ok = ok;
    if (error != null) result.error = error;
    return result;
  }

  PublishSessionKeyResponse._();

  factory PublishSessionKeyResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PublishSessionKeyResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PublishSessionKeyResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'ok')
    ..aOS(2, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishSessionKeyResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishSessionKeyResponse copyWith(
          void Function(PublishSessionKeyResponse) updates) =>
      super.copyWith((message) => updates(message as PublishSessionKeyResponse))
          as PublishSessionKeyResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PublishSessionKeyResponse create() => PublishSessionKeyResponse._();
  @$core.override
  PublishSessionKeyResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PublishSessionKeyResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PublishSessionKeyResponse>(create);
  static PublishSessionKeyResponse? _defaultInstance;

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

enum SettlementStreamMessage_Msg {
  hello,
  needPreSigs,
  providePreSigs,
  verifyOk,
  error,
  notSet
}

class SettlementStreamMessage extends $pb.GeneratedMessage {
  factory SettlementStreamMessage({
    SettlementHello? hello,
    NeedPreSigs? needPreSigs,
    ProvidePreSigs? providePreSigs,
    VerifyPreSigsOk? verifyOk,
    SettlementError? error,
  }) {
    final result = create();
    if (hello != null) result.hello = hello;
    if (needPreSigs != null) result.needPreSigs = needPreSigs;
    if (providePreSigs != null) result.providePreSigs = providePreSigs;
    if (verifyOk != null) result.verifyOk = verifyOk;
    if (error != null) result.error = error;
    return result;
  }

  SettlementStreamMessage._();

  factory SettlementStreamMessage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SettlementStreamMessage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, SettlementStreamMessage_Msg>
      _SettlementStreamMessage_MsgByTag = {
    1: SettlementStreamMessage_Msg.hello,
    2: SettlementStreamMessage_Msg.needPreSigs,
    3: SettlementStreamMessage_Msg.providePreSigs,
    4: SettlementStreamMessage_Msg.verifyOk,
    5: SettlementStreamMessage_Msg.error,
    0: SettlementStreamMessage_Msg.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SettlementStreamMessage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4, 5])
    ..aOM<SettlementHello>(1, _omitFieldNames ? '' : 'hello',
        subBuilder: SettlementHello.create)
    ..aOM<NeedPreSigs>(2, _omitFieldNames ? '' : 'needPreSigs',
        subBuilder: NeedPreSigs.create)
    ..aOM<ProvidePreSigs>(3, _omitFieldNames ? '' : 'providePreSigs',
        subBuilder: ProvidePreSigs.create)
    ..aOM<VerifyPreSigsOk>(4, _omitFieldNames ? '' : 'verifyOk',
        subBuilder: VerifyPreSigsOk.create)
    ..aOM<SettlementError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: SettlementError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementStreamMessage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementStreamMessage copyWith(
          void Function(SettlementStreamMessage) updates) =>
      super.copyWith((message) => updates(message as SettlementStreamMessage))
          as SettlementStreamMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SettlementStreamMessage create() => SettlementStreamMessage._();
  @$core.override
  SettlementStreamMessage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SettlementStreamMessage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SettlementStreamMessage>(create);
  static SettlementStreamMessage? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  SettlementStreamMessage_Msg whichMsg() =>
      _SettlementStreamMessage_MsgByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  void clearMsg() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  SettlementHello get hello => $_getN(0);
  @$pb.TagNumber(1)
  set hello(SettlementHello value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasHello() => $_has(0);
  @$pb.TagNumber(1)
  void clearHello() => $_clearField(1);
  @$pb.TagNumber(1)
  SettlementHello ensureHello() => $_ensure(0);

  @$pb.TagNumber(2)
  NeedPreSigs get needPreSigs => $_getN(1);
  @$pb.TagNumber(2)
  set needPreSigs(NeedPreSigs value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasNeedPreSigs() => $_has(1);
  @$pb.TagNumber(2)
  void clearNeedPreSigs() => $_clearField(2);
  @$pb.TagNumber(2)
  NeedPreSigs ensureNeedPreSigs() => $_ensure(1);

  @$pb.TagNumber(3)
  ProvidePreSigs get providePreSigs => $_getN(2);
  @$pb.TagNumber(3)
  set providePreSigs(ProvidePreSigs value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasProvidePreSigs() => $_has(2);
  @$pb.TagNumber(3)
  void clearProvidePreSigs() => $_clearField(3);
  @$pb.TagNumber(3)
  ProvidePreSigs ensureProvidePreSigs() => $_ensure(2);

  @$pb.TagNumber(4)
  VerifyPreSigsOk get verifyOk => $_getN(3);
  @$pb.TagNumber(4)
  set verifyOk(VerifyPreSigsOk value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasVerifyOk() => $_has(3);
  @$pb.TagNumber(4)
  void clearVerifyOk() => $_clearField(4);
  @$pb.TagNumber(4)
  VerifyPreSigsOk ensureVerifyOk() => $_ensure(3);

  @$pb.TagNumber(5)
  SettlementError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error(SettlementError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  SettlementError ensureError() => $_ensure(4);
}

/// Client → Server
class SettlementHello extends $pb.GeneratedMessage {
  factory SettlementHello({
    $core.String? matchId,
    $core.String? escrowId,
    $core.List<$core.int>? compPubkey,
    $core.String? token,
    $core.String? tableId,
    $core.String? sessionId,
    $core.int? seatIndex,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (escrowId != null) result.escrowId = escrowId;
    if (compPubkey != null) result.compPubkey = compPubkey;
    if (token != null) result.token = token;
    if (tableId != null) result.tableId = tableId;
    if (sessionId != null) result.sessionId = sessionId;
    if (seatIndex != null) result.seatIndex = seatIndex;
    return result;
  }

  SettlementHello._();

  factory SettlementHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SettlementHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SettlementHello',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aOS(2, _omitFieldNames ? '' : 'escrowId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'compPubkey', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'token')
    ..aOS(5, _omitFieldNames ? '' : 'tableId')
    ..aOS(6, _omitFieldNames ? '' : 'sessionId')
    ..aI(7, _omitFieldNames ? '' : 'seatIndex', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementHello copyWith(void Function(SettlementHello) updates) =>
      super.copyWith((message) => updates(message as SettlementHello))
          as SettlementHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SettlementHello create() => SettlementHello._();
  @$core.override
  SettlementHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SettlementHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SettlementHello>(create);
  static SettlementHello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get escrowId => $_getSZ(1);
  @$pb.TagNumber(2)
  set escrowId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEscrowId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEscrowId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get compPubkey => $_getN(2);
  @$pb.TagNumber(3)
  set compPubkey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCompPubkey() => $_has(2);
  @$pb.TagNumber(3)
  void clearCompPubkey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get token => $_getSZ(3);
  @$pb.TagNumber(4)
  set token($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasToken() => $_has(3);
  @$pb.TagNumber(4)
  void clearToken() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get tableId => $_getSZ(4);
  @$pb.TagNumber(5)
  set tableId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTableId() => $_has(4);
  @$pb.TagNumber(5)
  void clearTableId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get sessionId => $_getSZ(5);
  @$pb.TagNumber(6)
  set sessionId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSessionId() => $_has(5);
  @$pb.TagNumber(6)
  void clearSessionId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get seatIndex => $_getIZ(6);
  @$pb.TagNumber(7)
  set seatIndex($core.int value) => $_setUnsignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSeatIndex() => $_has(6);
  @$pb.TagNumber(7)
  void clearSeatIndex() => $_clearField(7);
}

/// Server → Client: request presigs for a specific branch/winner seat.
class NeedPreSigs extends $pb.GeneratedMessage {
  factory NeedPreSigs({
    $core.String? matchId,
    $core.int? branch,
    $core.String? draftTxHex,
    $core.Iterable<NeedPreSigsInput>? inputs,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (branch != null) result.branch = branch;
    if (draftTxHex != null) result.draftTxHex = draftTxHex;
    if (inputs != null) result.inputs.addAll(inputs);
    return result;
  }

  NeedPreSigs._();

  factory NeedPreSigs.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NeedPreSigs.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NeedPreSigs',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aI(2, _omitFieldNames ? '' : 'branch')
    ..aOS(3, _omitFieldNames ? '' : 'draftTxHex')
    ..pPM<NeedPreSigsInput>(4, _omitFieldNames ? '' : 'inputs',
        subBuilder: NeedPreSigsInput.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NeedPreSigs clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NeedPreSigs copyWith(void Function(NeedPreSigs) updates) =>
      super.copyWith((message) => updates(message as NeedPreSigs))
          as NeedPreSigs;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NeedPreSigs create() => NeedPreSigs._();
  @$core.override
  NeedPreSigs createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NeedPreSigs getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NeedPreSigs>(create);
  static NeedPreSigs? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get branch => $_getIZ(1);
  @$pb.TagNumber(2)
  set branch($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBranch() => $_has(1);
  @$pb.TagNumber(2)
  void clearBranch() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get draftTxHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set draftTxHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDraftTxHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearDraftTxHex() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<NeedPreSigsInput> get inputs => $_getList(3);
}

class NeedPreSigsInput extends $pb.GeneratedMessage {
  factory NeedPreSigsInput({
    $core.String? inputId,
    $core.String? redeemScriptHex,
    $core.String? sighashHex,
    $core.String? adaptorPointHex,
    $core.int? inputIndex,
    $fixnum.Int64? amountAtoms,
  }) {
    final result = create();
    if (inputId != null) result.inputId = inputId;
    if (redeemScriptHex != null) result.redeemScriptHex = redeemScriptHex;
    if (sighashHex != null) result.sighashHex = sighashHex;
    if (adaptorPointHex != null) result.adaptorPointHex = adaptorPointHex;
    if (inputIndex != null) result.inputIndex = inputIndex;
    if (amountAtoms != null) result.amountAtoms = amountAtoms;
    return result;
  }

  NeedPreSigsInput._();

  factory NeedPreSigsInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NeedPreSigsInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NeedPreSigsInput',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'inputId')
    ..aOS(2, _omitFieldNames ? '' : 'redeemScriptHex')
    ..aOS(3, _omitFieldNames ? '' : 'sighashHex')
    ..aOS(4, _omitFieldNames ? '' : 'adaptorPointHex')
    ..aI(5, _omitFieldNames ? '' : 'inputIndex', fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'amountAtoms', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NeedPreSigsInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NeedPreSigsInput copyWith(void Function(NeedPreSigsInput) updates) =>
      super.copyWith((message) => updates(message as NeedPreSigsInput))
          as NeedPreSigsInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NeedPreSigsInput create() => NeedPreSigsInput._();
  @$core.override
  NeedPreSigsInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NeedPreSigsInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NeedPreSigsInput>(create);
  static NeedPreSigsInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get inputId => $_getSZ(0);
  @$pb.TagNumber(1)
  set inputId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInputId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInputId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get redeemScriptHex => $_getSZ(1);
  @$pb.TagNumber(2)
  set redeemScriptHex($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRedeemScriptHex() => $_has(1);
  @$pb.TagNumber(2)
  void clearRedeemScriptHex() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sighashHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set sighashHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSighashHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearSighashHex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get adaptorPointHex => $_getSZ(3);
  @$pb.TagNumber(4)
  set adaptorPointHex($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAdaptorPointHex() => $_has(3);
  @$pb.TagNumber(4)
  void clearAdaptorPointHex() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get inputIndex => $_getIZ(4);
  @$pb.TagNumber(5)
  set inputIndex($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasInputIndex() => $_has(4);
  @$pb.TagNumber(5)
  void clearInputIndex() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get amountAtoms => $_getI64(5);
  @$pb.TagNumber(6)
  set amountAtoms($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAmountAtoms() => $_has(5);
  @$pb.TagNumber(6)
  void clearAmountAtoms() => $_clearField(6);
}

/// Client → Server: presigs for the requested branch/input set.
class ProvidePreSigs extends $pb.GeneratedMessage {
  factory ProvidePreSigs({
    $core.String? matchId,
    $core.int? branch,
    $core.Iterable<PreSignature>? presigs,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (branch != null) result.branch = branch;
    if (presigs != null) result.presigs.addAll(presigs);
    return result;
  }

  ProvidePreSigs._();

  factory ProvidePreSigs.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvidePreSigs.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvidePreSigs',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aI(2, _omitFieldNames ? '' : 'branch')
    ..pPM<PreSignature>(3, _omitFieldNames ? '' : 'presigs',
        subBuilder: PreSignature.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvidePreSigs clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvidePreSigs copyWith(void Function(ProvidePreSigs) updates) =>
      super.copyWith((message) => updates(message as ProvidePreSigs))
          as ProvidePreSigs;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvidePreSigs create() => ProvidePreSigs._();
  @$core.override
  ProvidePreSigs createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvidePreSigs getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvidePreSigs>(create);
  static ProvidePreSigs? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get branch => $_getIZ(1);
  @$pb.TagNumber(2)
  set branch($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBranch() => $_has(1);
  @$pb.TagNumber(2)
  void clearBranch() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<PreSignature> get presigs => $_getList(2);
}

class PreSignature extends $pb.GeneratedMessage {
  factory PreSignature({
    $core.String? inputId,
    $core.String? rPrimeCompactHex,
    $core.String? sPrimeHex,
  }) {
    final result = create();
    if (inputId != null) result.inputId = inputId;
    if (rPrimeCompactHex != null) result.rPrimeCompactHex = rPrimeCompactHex;
    if (sPrimeHex != null) result.sPrimeHex = sPrimeHex;
    return result;
  }

  PreSignature._();

  factory PreSignature.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PreSignature.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PreSignature',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'inputId')
    ..aOS(2, _omitFieldNames ? '' : 'rPrimeCompactHex')
    ..aOS(3, _omitFieldNames ? '' : 'sPrimeHex')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PreSignature clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PreSignature copyWith(void Function(PreSignature) updates) =>
      super.copyWith((message) => updates(message as PreSignature))
          as PreSignature;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PreSignature create() => PreSignature._();
  @$core.override
  PreSignature createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PreSignature getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PreSignature>(create);
  static PreSignature? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get inputId => $_getSZ(0);
  @$pb.TagNumber(1)
  set inputId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInputId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInputId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get rPrimeCompactHex => $_getSZ(1);
  @$pb.TagNumber(2)
  set rPrimeCompactHex($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRPrimeCompactHex() => $_has(1);
  @$pb.TagNumber(2)
  void clearRPrimeCompactHex() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sPrimeHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set sPrimeHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSPrimeHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearSPrimeHex() => $_clearField(3);
}

/// Server → Client: indicates presigs were verified and stored.
class VerifyPreSigsOk extends $pb.GeneratedMessage {
  factory VerifyPreSigsOk({
    $core.String? matchId,
    $core.int? branch,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (branch != null) result.branch = branch;
    return result;
  }

  VerifyPreSigsOk._();

  factory VerifyPreSigsOk.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory VerifyPreSigsOk.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'VerifyPreSigsOk',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aI(2, _omitFieldNames ? '' : 'branch')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VerifyPreSigsOk clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VerifyPreSigsOk copyWith(void Function(VerifyPreSigsOk) updates) =>
      super.copyWith((message) => updates(message as VerifyPreSigsOk))
          as VerifyPreSigsOk;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static VerifyPreSigsOk create() => VerifyPreSigsOk._();
  @$core.override
  VerifyPreSigsOk createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static VerifyPreSigsOk getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<VerifyPreSigsOk>(create);
  static VerifyPreSigsOk? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get branch => $_getIZ(1);
  @$pb.TagNumber(2)
  set branch($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBranch() => $_has(1);
  @$pb.TagNumber(2)
  void clearBranch() => $_clearField(2);
}

/// Server → Client: terminal error for this stream.
class SettlementError extends $pb.GeneratedMessage {
  factory SettlementError({
    $core.String? matchId,
    $core.String? error,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (error != null) result.error = error;
    return result;
  }

  SettlementError._();

  factory SettlementError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SettlementError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SettlementError',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aOS(2, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SettlementError copyWith(void Function(SettlementError) updates) =>
      super.copyWith((message) => updates(message as SettlementError))
          as SettlementError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SettlementError create() => SettlementError._();
  @$core.override
  SettlementError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SettlementError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SettlementError>(create);
  static SettlementError? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get error => $_getSZ(1);
  @$pb.TagNumber(2)
  set error($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
}

class GetFinalizeBundleRequest extends $pb.GeneratedMessage {
  factory GetFinalizeBundleRequest({
    $core.String? matchId,
    $core.int? winnerSeat,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (winnerSeat != null) result.winnerSeat = winnerSeat;
    return result;
  }

  GetFinalizeBundleRequest._();

  factory GetFinalizeBundleRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetFinalizeBundleRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetFinalizeBundleRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aI(2, _omitFieldNames ? '' : 'winnerSeat')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetFinalizeBundleRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetFinalizeBundleRequest copyWith(
          void Function(GetFinalizeBundleRequest) updates) =>
      super.copyWith((message) => updates(message as GetFinalizeBundleRequest))
          as GetFinalizeBundleRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetFinalizeBundleRequest create() => GetFinalizeBundleRequest._();
  @$core.override
  GetFinalizeBundleRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetFinalizeBundleRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetFinalizeBundleRequest>(create);
  static GetFinalizeBundleRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get winnerSeat => $_getIZ(1);
  @$pb.TagNumber(2)
  set winnerSeat($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWinnerSeat() => $_has(1);
  @$pb.TagNumber(2)
  void clearWinnerSeat() => $_clearField(2);
}

class GetFinalizeBundleResponse extends $pb.GeneratedMessage {
  factory GetFinalizeBundleResponse({
    $core.String? matchId,
    $core.int? branch,
    $core.String? draftTxHex,
    $core.String? gammaHex,
    $core.Iterable<FinalizeInput>? inputs,
  }) {
    final result = create();
    if (matchId != null) result.matchId = matchId;
    if (branch != null) result.branch = branch;
    if (draftTxHex != null) result.draftTxHex = draftTxHex;
    if (gammaHex != null) result.gammaHex = gammaHex;
    if (inputs != null) result.inputs.addAll(inputs);
    return result;
  }

  GetFinalizeBundleResponse._();

  factory GetFinalizeBundleResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetFinalizeBundleResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetFinalizeBundleResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'matchId')
    ..aI(2, _omitFieldNames ? '' : 'branch')
    ..aOS(3, _omitFieldNames ? '' : 'draftTxHex')
    ..aOS(4, _omitFieldNames ? '' : 'gammaHex')
    ..pPM<FinalizeInput>(5, _omitFieldNames ? '' : 'inputs',
        subBuilder: FinalizeInput.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetFinalizeBundleResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetFinalizeBundleResponse copyWith(
          void Function(GetFinalizeBundleResponse) updates) =>
      super.copyWith((message) => updates(message as GetFinalizeBundleResponse))
          as GetFinalizeBundleResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetFinalizeBundleResponse create() => GetFinalizeBundleResponse._();
  @$core.override
  GetFinalizeBundleResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetFinalizeBundleResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetFinalizeBundleResponse>(create);
  static GetFinalizeBundleResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get matchId => $_getSZ(0);
  @$pb.TagNumber(1)
  set matchId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMatchId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMatchId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get branch => $_getIZ(1);
  @$pb.TagNumber(2)
  set branch($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBranch() => $_has(1);
  @$pb.TagNumber(2)
  void clearBranch() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get draftTxHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set draftTxHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDraftTxHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearDraftTxHex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get gammaHex => $_getSZ(3);
  @$pb.TagNumber(4)
  set gammaHex($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasGammaHex() => $_has(3);
  @$pb.TagNumber(4)
  void clearGammaHex() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<FinalizeInput> get inputs => $_getList(4);
}

class FinalizeInput extends $pb.GeneratedMessage {
  factory FinalizeInput({
    $core.String? inputId,
    $core.String? rPrimeCompactHex,
    $core.String? sPrimeHex,
    $core.int? inputIndex,
    $core.String? redeemScriptHex,
  }) {
    final result = create();
    if (inputId != null) result.inputId = inputId;
    if (rPrimeCompactHex != null) result.rPrimeCompactHex = rPrimeCompactHex;
    if (sPrimeHex != null) result.sPrimeHex = sPrimeHex;
    if (inputIndex != null) result.inputIndex = inputIndex;
    if (redeemScriptHex != null) result.redeemScriptHex = redeemScriptHex;
    return result;
  }

  FinalizeInput._();

  factory FinalizeInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FinalizeInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FinalizeInput',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'poker'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'inputId')
    ..aOS(2, _omitFieldNames ? '' : 'rPrimeCompactHex')
    ..aOS(3, _omitFieldNames ? '' : 'sPrimeHex')
    ..aI(4, _omitFieldNames ? '' : 'inputIndex', fieldType: $pb.PbFieldType.OU3)
    ..aOS(5, _omitFieldNames ? '' : 'redeemScriptHex')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FinalizeInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FinalizeInput copyWith(void Function(FinalizeInput) updates) =>
      super.copyWith((message) => updates(message as FinalizeInput))
          as FinalizeInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FinalizeInput create() => FinalizeInput._();
  @$core.override
  FinalizeInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FinalizeInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FinalizeInput>(create);
  static FinalizeInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get inputId => $_getSZ(0);
  @$pb.TagNumber(1)
  set inputId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInputId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInputId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get rPrimeCompactHex => $_getSZ(1);
  @$pb.TagNumber(2)
  set rPrimeCompactHex($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRPrimeCompactHex() => $_has(1);
  @$pb.TagNumber(2)
  void clearRPrimeCompactHex() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sPrimeHex => $_getSZ(2);
  @$pb.TagNumber(3)
  set sPrimeHex($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSPrimeHex() => $_has(2);
  @$pb.TagNumber(3)
  void clearSPrimeHex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get inputIndex => $_getIZ(3);
  @$pb.TagNumber(4)
  set inputIndex($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasInputIndex() => $_has(3);
  @$pb.TagNumber(4)
  void clearInputIndex() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get redeemScriptHex => $_getSZ(4);
  @$pb.TagNumber(5)
  set redeemScriptHex($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRedeemScriptHex() => $_has(4);
  @$pb.TagNumber(5)
  void clearRedeemScriptHex() => $_clearField(5);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
