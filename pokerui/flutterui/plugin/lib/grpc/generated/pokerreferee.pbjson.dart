// This is a generated file - do not edit.
//
// Generated from pokerreferee.proto.

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

@$core.Deprecated('Use openEscrowRequestDescriptor instead')
const OpenEscrowRequest$json = {
  '1': 'OpenEscrowRequest',
  '2': [
    {'1': 'amount_atoms', '3': 1, '4': 1, '5': 4, '10': 'amountAtoms'},
    {'1': 'csv_blocks', '3': 2, '4': 1, '5': 13, '10': 'csvBlocks'},
    {'1': 'token', '3': 3, '4': 1, '5': 9, '10': 'token'},
    {'1': 'comp_pubkey', '3': 4, '4': 1, '5': 12, '10': 'compPubkey'},
  ],
};

/// Descriptor for `OpenEscrowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openEscrowRequestDescriptor = $convert.base64Decode(
    'ChFPcGVuRXNjcm93UmVxdWVzdBIhCgxhbW91bnRfYXRvbXMYASABKARSC2Ftb3VudEF0b21zEh'
    '0KCmNzdl9ibG9ja3MYAiABKA1SCWNzdkJsb2NrcxIUCgV0b2tlbhgDIAEoCVIFdG9rZW4SHwoL'
    'Y29tcF9wdWJrZXkYBCABKAxSCmNvbXBQdWJrZXk=');

@$core.Deprecated('Use openEscrowResponseDescriptor instead')
const OpenEscrowResponse$json = {
  '1': 'OpenEscrowResponse',
  '2': [
    {'1': 'escrow_id', '3': 1, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'deposit_addr', '3': 2, '4': 1, '5': 9, '10': 'depositAddr'},
    {'1': 'redeem_script_hex', '3': 3, '4': 1, '5': 9, '10': 'redeemScriptHex'},
    {'1': 'pk_script_hex', '3': 4, '4': 1, '5': 9, '10': 'pkScriptHex'},
    {'1': 'match_id', '3': 5, '4': 1, '5': 9, '10': 'matchId'},
    {
      '1': 'required_confirmations',
      '3': 6,
      '4': 1,
      '5': 13,
      '10': 'requiredConfirmations'
    },
  ],
};

/// Descriptor for `OpenEscrowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openEscrowResponseDescriptor = $convert.base64Decode(
    'ChJPcGVuRXNjcm93UmVzcG9uc2USGwoJZXNjcm93X2lkGAEgASgJUghlc2Nyb3dJZBIhCgxkZX'
    'Bvc2l0X2FkZHIYAiABKAlSC2RlcG9zaXRBZGRyEioKEXJlZGVlbV9zY3JpcHRfaGV4GAMgASgJ'
    'Ug9yZWRlZW1TY3JpcHRIZXgSIgoNcGtfc2NyaXB0X2hleBgEIAEoCVILcGtTY3JpcHRIZXgSGQ'
    'oIbWF0Y2hfaWQYBSABKAlSB21hdGNoSWQSNQoWcmVxdWlyZWRfY29uZmlybWF0aW9ucxgGIAEo'
    'DVIVcmVxdWlyZWRDb25maXJtYXRpb25z');

@$core.Deprecated('Use bindEscrowRequestDescriptor instead')
const BindEscrowRequest$json = {
  '1': 'BindEscrowRequest',
  '2': [
    {'1': 'outpoint', '3': 1, '4': 1, '5': 9, '10': 'outpoint'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'session_id', '3': 3, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'seat_index', '3': 4, '4': 1, '5': 13, '10': 'seatIndex'},
    {'1': 'match_id', '3': 5, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'token', '3': 6, '4': 1, '5': 9, '10': 'token'},
    {'1': 'redeem_script_hex', '3': 7, '4': 1, '5': 9, '10': 'redeemScriptHex'},
    {'1': 'csv_blocks', '3': 8, '4': 1, '5': 13, '10': 'csvBlocks'},
  ],
};

/// Descriptor for `BindEscrowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bindEscrowRequestDescriptor = $convert.base64Decode(
    'ChFCaW5kRXNjcm93UmVxdWVzdBIaCghvdXRwb2ludBgBIAEoCVIIb3V0cG9pbnQSGQoIdGFibG'
    'VfaWQYAiABKAlSB3RhYmxlSWQSHQoKc2Vzc2lvbl9pZBgDIAEoCVIJc2Vzc2lvbklkEh0KCnNl'
    'YXRfaW5kZXgYBCABKA1SCXNlYXRJbmRleBIZCghtYXRjaF9pZBgFIAEoCVIHbWF0Y2hJZBIUCg'
    'V0b2tlbhgGIAEoCVIFdG9rZW4SKgoRcmVkZWVtX3NjcmlwdF9oZXgYByABKAlSD3JlZGVlbVNj'
    'cmlwdEhleBIdCgpjc3ZfYmxvY2tzGAggASgNUgljc3ZCbG9ja3M=');

@$core.Deprecated('Use bindEscrowResponseDescriptor instead')
const BindEscrowResponse$json = {
  '1': 'BindEscrowResponse',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'table_id', '3': 2, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'session_id', '3': 3, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'seat_index', '3': 4, '4': 1, '5': 13, '10': 'seatIndex'},
    {'1': 'escrow_id', '3': 5, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'escrow_ready', '3': 6, '4': 1, '5': 8, '10': 'escrowReady'},
    {'1': 'amount_atoms', '3': 7, '4': 1, '5': 4, '10': 'amountAtoms'},
    {
      '1': 'required_amount_atoms',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'requiredAmountAtoms'
    },
    {'1': 'outpoint', '3': 9, '4': 1, '5': 9, '10': 'outpoint'},
  ],
};

/// Descriptor for `BindEscrowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bindEscrowResponseDescriptor = $convert.base64Decode(
    'ChJCaW5kRXNjcm93UmVzcG9uc2USGQoIbWF0Y2hfaWQYASABKAlSB21hdGNoSWQSGQoIdGFibG'
    'VfaWQYAiABKAlSB3RhYmxlSWQSHQoKc2Vzc2lvbl9pZBgDIAEoCVIJc2Vzc2lvbklkEh0KCnNl'
    'YXRfaW5kZXgYBCABKA1SCXNlYXRJbmRleBIbCgllc2Nyb3dfaWQYBSABKAlSCGVzY3Jvd0lkEi'
    'EKDGVzY3Jvd19yZWFkeRgGIAEoCFILZXNjcm93UmVhZHkSIQoMYW1vdW50X2F0b21zGAcgASgE'
    'UgthbW91bnRBdG9tcxIyChVyZXF1aXJlZF9hbW91bnRfYXRvbXMYCCABKARSE3JlcXVpcmVkQW'
    '1vdW50QXRvbXMSGgoIb3V0cG9pbnQYCSABKAlSCG91dHBvaW50');

@$core.Deprecated('Use getEscrowStatusRequestDescriptor instead')
const GetEscrowStatusRequest$json = {
  '1': 'GetEscrowStatusRequest',
  '2': [
    {'1': 'escrow_id', '3': 1, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'token', '3': 2, '4': 1, '5': 9, '10': 'token'},
  ],
};

/// Descriptor for `GetEscrowStatusRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getEscrowStatusRequestDescriptor =
    $convert.base64Decode(
        'ChZHZXRFc2Nyb3dTdGF0dXNSZXF1ZXN0EhsKCWVzY3Jvd19pZBgBIAEoCVIIZXNjcm93SWQSFA'
        'oFdG9rZW4YAiABKAlSBXRva2Vu');

@$core.Deprecated('Use getEscrowStatusResponseDescriptor instead')
const GetEscrowStatusResponse$json = {
  '1': 'GetEscrowStatusResponse',
  '2': [
    {'1': 'escrow_id', '3': 1, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'confs', '3': 2, '4': 1, '5': 13, '10': 'confs'},
    {'1': 'utxo_count', '3': 3, '4': 1, '5': 13, '10': 'utxoCount'},
    {'1': 'ok', '3': 4, '4': 1, '5': 8, '10': 'ok'},
    {'1': 'updated_at_unix', '3': 5, '4': 1, '5': 3, '10': 'updatedAtUnix'},
    {'1': 'funding_txid', '3': 6, '4': 1, '5': 9, '10': 'fundingTxid'},
    {'1': 'funding_vout', '3': 7, '4': 1, '5': 13, '10': 'fundingVout'},
    {'1': 'amount_atoms', '3': 8, '4': 1, '5': 4, '10': 'amountAtoms'},
    {'1': 'pk_script_hex', '3': 9, '4': 1, '5': 9, '10': 'pkScriptHex'},
    {'1': 'csv_blocks', '3': 10, '4': 1, '5': 13, '10': 'csvBlocks'},
    {
      '1': 'required_confirmations',
      '3': 11,
      '4': 1,
      '5': 13,
      '10': 'requiredConfirmations'
    },
    {'1': 'mature_for_csv', '3': 12, '4': 1, '5': 8, '10': 'matureForCsv'},
  ],
};

/// Descriptor for `GetEscrowStatusResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getEscrowStatusResponseDescriptor = $convert.base64Decode(
    'ChdHZXRFc2Nyb3dTdGF0dXNSZXNwb25zZRIbCgllc2Nyb3dfaWQYASABKAlSCGVzY3Jvd0lkEh'
    'QKBWNvbmZzGAIgASgNUgVjb25mcxIdCgp1dHhvX2NvdW50GAMgASgNUgl1dHhvQ291bnQSDgoC'
    'b2sYBCABKAhSAm9rEiYKD3VwZGF0ZWRfYXRfdW5peBgFIAEoA1INdXBkYXRlZEF0VW5peBIhCg'
    'xmdW5kaW5nX3R4aWQYBiABKAlSC2Z1bmRpbmdUeGlkEiEKDGZ1bmRpbmdfdm91dBgHIAEoDVIL'
    'ZnVuZGluZ1ZvdXQSIQoMYW1vdW50X2F0b21zGAggASgEUgthbW91bnRBdG9tcxIiCg1wa19zY3'
    'JpcHRfaGV4GAkgASgJUgtwa1NjcmlwdEhleBIdCgpjc3ZfYmxvY2tzGAogASgNUgljc3ZCbG9j'
    'a3MSNQoWcmVxdWlyZWRfY29uZmlybWF0aW9ucxgLIAEoDVIVcmVxdWlyZWRDb25maXJtYXRpb2'
    '5zEiQKDm1hdHVyZV9mb3JfY3N2GAwgASgIUgxtYXR1cmVGb3JDc3Y=');

@$core.Deprecated('Use publishSessionKeyRequestDescriptor instead')
const PublishSessionKeyRequest$json = {
  '1': 'PublishSessionKeyRequest',
  '2': [
    {'1': 'escrow_id', '3': 1, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'comp_pubkey', '3': 2, '4': 1, '5': 12, '10': 'compPubkey'},
  ],
};

/// Descriptor for `PublishSessionKeyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List publishSessionKeyRequestDescriptor =
    $convert.base64Decode(
        'ChhQdWJsaXNoU2Vzc2lvbktleVJlcXVlc3QSGwoJZXNjcm93X2lkGAEgASgJUghlc2Nyb3dJZB'
        'IfCgtjb21wX3B1YmtleRgCIAEoDFIKY29tcFB1YmtleQ==');

@$core.Deprecated('Use publishSessionKeyResponseDescriptor instead')
const PublishSessionKeyResponse$json = {
  '1': 'PublishSessionKeyResponse',
  '2': [
    {'1': 'ok', '3': 1, '4': 1, '5': 8, '10': 'ok'},
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `PublishSessionKeyResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List publishSessionKeyResponseDescriptor =
    $convert.base64Decode(
        'ChlQdWJsaXNoU2Vzc2lvbktleVJlc3BvbnNlEg4KAm9rGAEgASgIUgJvaxIUCgVlcnJvchgCIA'
        'EoCVIFZXJyb3I=');

@$core.Deprecated('Use settlementStreamMessageDescriptor instead')
const SettlementStreamMessage$json = {
  '1': 'SettlementStreamMessage',
  '2': [
    {
      '1': 'hello',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.poker.SettlementHello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'need_pre_sigs',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.poker.NeedPreSigs',
      '9': 0,
      '10': 'needPreSigs'
    },
    {
      '1': 'provide_pre_sigs',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.poker.ProvidePreSigs',
      '9': 0,
      '10': 'providePreSigs'
    },
    {
      '1': 'verify_ok',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.poker.VerifyPreSigsOk',
      '9': 0,
      '10': 'verifyOk'
    },
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.poker.SettlementError',
      '9': 0,
      '10': 'error'
    },
  ],
  '8': [
    {'1': 'msg'},
  ],
};

/// Descriptor for `SettlementStreamMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List settlementStreamMessageDescriptor = $convert.base64Decode(
    'ChdTZXR0bGVtZW50U3RyZWFtTWVzc2FnZRIuCgVoZWxsbxgBIAEoCzIWLnBva2VyLlNldHRsZW'
    '1lbnRIZWxsb0gAUgVoZWxsbxI4Cg1uZWVkX3ByZV9zaWdzGAIgASgLMhIucG9rZXIuTmVlZFBy'
    'ZVNpZ3NIAFILbmVlZFByZVNpZ3MSQQoQcHJvdmlkZV9wcmVfc2lncxgDIAEoCzIVLnBva2VyLl'
    'Byb3ZpZGVQcmVTaWdzSABSDnByb3ZpZGVQcmVTaWdzEjUKCXZlcmlmeV9vaxgEIAEoCzIWLnBv'
    'a2VyLlZlcmlmeVByZVNpZ3NPa0gAUgh2ZXJpZnlPaxIuCgVlcnJvchgFIAEoCzIWLnBva2VyLl'
    'NldHRsZW1lbnRFcnJvckgAUgVlcnJvckIFCgNtc2c=');

@$core.Deprecated('Use settlementHelloDescriptor instead')
const SettlementHello$json = {
  '1': 'SettlementHello',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'escrow_id', '3': 2, '4': 1, '5': 9, '10': 'escrowId'},
    {'1': 'comp_pubkey', '3': 3, '4': 1, '5': 12, '10': 'compPubkey'},
    {'1': 'token', '3': 4, '4': 1, '5': 9, '10': 'token'},
    {'1': 'table_id', '3': 5, '4': 1, '5': 9, '10': 'tableId'},
    {'1': 'session_id', '3': 6, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'seat_index', '3': 7, '4': 1, '5': 13, '10': 'seatIndex'},
  ],
};

/// Descriptor for `SettlementHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List settlementHelloDescriptor = $convert.base64Decode(
    'Cg9TZXR0bGVtZW50SGVsbG8SGQoIbWF0Y2hfaWQYASABKAlSB21hdGNoSWQSGwoJZXNjcm93X2'
    'lkGAIgASgJUghlc2Nyb3dJZBIfCgtjb21wX3B1YmtleRgDIAEoDFIKY29tcFB1YmtleRIUCgV0'
    'b2tlbhgEIAEoCVIFdG9rZW4SGQoIdGFibGVfaWQYBSABKAlSB3RhYmxlSWQSHQoKc2Vzc2lvbl'
    '9pZBgGIAEoCVIJc2Vzc2lvbklkEh0KCnNlYXRfaW5kZXgYByABKA1SCXNlYXRJbmRleA==');

@$core.Deprecated('Use needPreSigsDescriptor instead')
const NeedPreSigs$json = {
  '1': 'NeedPreSigs',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'branch', '3': 2, '4': 1, '5': 5, '10': 'branch'},
    {'1': 'draft_tx_hex', '3': 3, '4': 1, '5': 9, '10': 'draftTxHex'},
    {
      '1': 'inputs',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.poker.NeedPreSigsInput',
      '10': 'inputs'
    },
  ],
};

/// Descriptor for `NeedPreSigs`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List needPreSigsDescriptor = $convert.base64Decode(
    'CgtOZWVkUHJlU2lncxIZCghtYXRjaF9pZBgBIAEoCVIHbWF0Y2hJZBIWCgZicmFuY2gYAiABKA'
    'VSBmJyYW5jaBIgCgxkcmFmdF90eF9oZXgYAyABKAlSCmRyYWZ0VHhIZXgSLwoGaW5wdXRzGAQg'
    'AygLMhcucG9rZXIuTmVlZFByZVNpZ3NJbnB1dFIGaW5wdXRz');

@$core.Deprecated('Use needPreSigsInputDescriptor instead')
const NeedPreSigsInput$json = {
  '1': 'NeedPreSigsInput',
  '2': [
    {'1': 'input_id', '3': 1, '4': 1, '5': 9, '10': 'inputId'},
    {'1': 'redeem_script_hex', '3': 2, '4': 1, '5': 9, '10': 'redeemScriptHex'},
    {'1': 'sighash_hex', '3': 3, '4': 1, '5': 9, '10': 'sighashHex'},
    {'1': 'adaptor_point_hex', '3': 4, '4': 1, '5': 9, '10': 'adaptorPointHex'},
    {'1': 'input_index', '3': 5, '4': 1, '5': 13, '10': 'inputIndex'},
    {'1': 'amount_atoms', '3': 6, '4': 1, '5': 4, '10': 'amountAtoms'},
  ],
};

/// Descriptor for `NeedPreSigsInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List needPreSigsInputDescriptor = $convert.base64Decode(
    'ChBOZWVkUHJlU2lnc0lucHV0EhkKCGlucHV0X2lkGAEgASgJUgdpbnB1dElkEioKEXJlZGVlbV'
    '9zY3JpcHRfaGV4GAIgASgJUg9yZWRlZW1TY3JpcHRIZXgSHwoLc2lnaGFzaF9oZXgYAyABKAlS'
    'CnNpZ2hhc2hIZXgSKgoRYWRhcHRvcl9wb2ludF9oZXgYBCABKAlSD2FkYXB0b3JQb2ludEhleB'
    'IfCgtpbnB1dF9pbmRleBgFIAEoDVIKaW5wdXRJbmRleBIhCgxhbW91bnRfYXRvbXMYBiABKARS'
    'C2Ftb3VudEF0b21z');

@$core.Deprecated('Use providePreSigsDescriptor instead')
const ProvidePreSigs$json = {
  '1': 'ProvidePreSigs',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'branch', '3': 2, '4': 1, '5': 5, '10': 'branch'},
    {
      '1': 'presigs',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.poker.PreSignature',
      '10': 'presigs'
    },
  ],
};

/// Descriptor for `ProvidePreSigs`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List providePreSigsDescriptor = $convert.base64Decode(
    'Cg5Qcm92aWRlUHJlU2lncxIZCghtYXRjaF9pZBgBIAEoCVIHbWF0Y2hJZBIWCgZicmFuY2gYAi'
    'ABKAVSBmJyYW5jaBItCgdwcmVzaWdzGAMgAygLMhMucG9rZXIuUHJlU2lnbmF0dXJlUgdwcmVz'
    'aWdz');

@$core.Deprecated('Use preSignatureDescriptor instead')
const PreSignature$json = {
  '1': 'PreSignature',
  '2': [
    {'1': 'input_id', '3': 1, '4': 1, '5': 9, '10': 'inputId'},
    {
      '1': 'r_prime_compact_hex',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'rPrimeCompactHex'
    },
    {'1': 's_prime_hex', '3': 3, '4': 1, '5': 9, '10': 'sPrimeHex'},
  ],
};

/// Descriptor for `PreSignature`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List preSignatureDescriptor = $convert.base64Decode(
    'CgxQcmVTaWduYXR1cmUSGQoIaW5wdXRfaWQYASABKAlSB2lucHV0SWQSLQoTcl9wcmltZV9jb2'
    '1wYWN0X2hleBgCIAEoCVIQclByaW1lQ29tcGFjdEhleBIeCgtzX3ByaW1lX2hleBgDIAEoCVIJ'
    'c1ByaW1lSGV4');

@$core.Deprecated('Use verifyPreSigsOkDescriptor instead')
const VerifyPreSigsOk$json = {
  '1': 'VerifyPreSigsOk',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'branch', '3': 2, '4': 1, '5': 5, '10': 'branch'},
  ],
};

/// Descriptor for `VerifyPreSigsOk`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List verifyPreSigsOkDescriptor = $convert.base64Decode(
    'Cg9WZXJpZnlQcmVTaWdzT2sSGQoIbWF0Y2hfaWQYASABKAlSB21hdGNoSWQSFgoGYnJhbmNoGA'
    'IgASgFUgZicmFuY2g=');

@$core.Deprecated('Use settlementErrorDescriptor instead')
const SettlementError$json = {
  '1': 'SettlementError',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `SettlementError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List settlementErrorDescriptor = $convert.base64Decode(
    'Cg9TZXR0bGVtZW50RXJyb3ISGQoIbWF0Y2hfaWQYASABKAlSB21hdGNoSWQSFAoFZXJyb3IYAi'
    'ABKAlSBWVycm9y');

@$core.Deprecated('Use getFinalizeBundleRequestDescriptor instead')
const GetFinalizeBundleRequest$json = {
  '1': 'GetFinalizeBundleRequest',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'winner_seat', '3': 2, '4': 1, '5': 5, '10': 'winnerSeat'},
  ],
};

/// Descriptor for `GetFinalizeBundleRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getFinalizeBundleRequestDescriptor =
    $convert.base64Decode(
        'ChhHZXRGaW5hbGl6ZUJ1bmRsZVJlcXVlc3QSGQoIbWF0Y2hfaWQYASABKAlSB21hdGNoSWQSHw'
        'oLd2lubmVyX3NlYXQYAiABKAVSCndpbm5lclNlYXQ=');

@$core.Deprecated('Use getFinalizeBundleResponseDescriptor instead')
const GetFinalizeBundleResponse$json = {
  '1': 'GetFinalizeBundleResponse',
  '2': [
    {'1': 'match_id', '3': 1, '4': 1, '5': 9, '10': 'matchId'},
    {'1': 'branch', '3': 2, '4': 1, '5': 5, '10': 'branch'},
    {'1': 'draft_tx_hex', '3': 3, '4': 1, '5': 9, '10': 'draftTxHex'},
    {'1': 'gamma_hex', '3': 4, '4': 1, '5': 9, '10': 'gammaHex'},
    {
      '1': 'inputs',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.poker.FinalizeInput',
      '10': 'inputs'
    },
  ],
};

/// Descriptor for `GetFinalizeBundleResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getFinalizeBundleResponseDescriptor = $convert.base64Decode(
    'ChlHZXRGaW5hbGl6ZUJ1bmRsZVJlc3BvbnNlEhkKCG1hdGNoX2lkGAEgASgJUgdtYXRjaElkEh'
    'YKBmJyYW5jaBgCIAEoBVIGYnJhbmNoEiAKDGRyYWZ0X3R4X2hleBgDIAEoCVIKZHJhZnRUeEhl'
    'eBIbCglnYW1tYV9oZXgYBCABKAlSCGdhbW1hSGV4EiwKBmlucHV0cxgFIAMoCzIULnBva2VyLk'
    'ZpbmFsaXplSW5wdXRSBmlucHV0cw==');

@$core.Deprecated('Use finalizeInputDescriptor instead')
const FinalizeInput$json = {
  '1': 'FinalizeInput',
  '2': [
    {'1': 'input_id', '3': 1, '4': 1, '5': 9, '10': 'inputId'},
    {
      '1': 'r_prime_compact_hex',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'rPrimeCompactHex'
    },
    {'1': 's_prime_hex', '3': 3, '4': 1, '5': 9, '10': 'sPrimeHex'},
    {'1': 'input_index', '3': 4, '4': 1, '5': 13, '10': 'inputIndex'},
    {'1': 'redeem_script_hex', '3': 5, '4': 1, '5': 9, '10': 'redeemScriptHex'},
  ],
};

/// Descriptor for `FinalizeInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List finalizeInputDescriptor = $convert.base64Decode(
    'Cg1GaW5hbGl6ZUlucHV0EhkKCGlucHV0X2lkGAEgASgJUgdpbnB1dElkEi0KE3JfcHJpbWVfY2'
    '9tcGFjdF9oZXgYAiABKAlSEHJQcmltZUNvbXBhY3RIZXgSHgoLc19wcmltZV9oZXgYAyABKAlS'
    'CXNQcmltZUhleBIfCgtpbnB1dF9pbmRleBgEIAEoDVIKaW5wdXRJbmRleBIqChFyZWRlZW1fc2'
    'NyaXB0X2hleBgFIAEoCVIPcmVkZWVtU2NyaXB0SGV4');

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
