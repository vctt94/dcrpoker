// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'definitions.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

InitClient _$InitClientFromJson(Map<String, dynamic> json) => InitClient(
      json['server_addr'] as String,
      json['grpc_cert_path'] as String,
      json['datadir'] as String,
      json['payout_address'] as String,
      json['log_file'] as String,
      json['debug_level'] as String,
      json['wants_log_ntfns'] as bool,
      json['rpc_websocket_url'] as String,
      json['rpc_cert_path'] as String,
      json['rpc_client_cert_path'] as String,
      json['rpc_client_key_path'] as String,
      json['rpc_user'] as String,
      json['rpc_pass'] as String,
    );

Map<String, dynamic> _$InitClientToJson(InitClient instance) =>
    <String, dynamic>{
      'server_addr': instance.serverAddr,
      'grpc_cert_path': instance.grpcCertPath,
      'datadir': instance.dataDir,
      'payout_address': instance.payoutAddress,
      'log_file': instance.logFile,
      'debug_level': instance.debugLevel,
      'wants_log_ntfns': instance.wantsLogNtfns,
      'rpc_websocket_url': instance.rpcWebsockeURL,
      'rpc_cert_path': instance.rpcCertPath,
      'rpc_client_cert_path': instance.rpcClientCertpath,
      'rpc_client_key_path': instance.rpcClientKeypath,
      'rpc_user': instance.rpcUser,
      'rpc_pass': instance.rpcPass,
    };

InitPokerClient _$InitPokerClientFromJson(Map<String, dynamic> json) =>
    InitPokerClient(
      json['datadir'] as String,
      json['grpc_host'] as String,
      json['grpc_port'] as String,
      json['grpc_server_cert'] as String,
      json['insecure'] as bool,
      json['offline'] as bool,
      json['player_id'] as String?,
      json['log_file'] as String,
      json['debug_level'] as String,
    );

Map<String, dynamic> _$InitPokerClientToJson(InitPokerClient instance) =>
    <String, dynamic>{
      'datadir': instance.dataDir,
      'grpc_host': instance.grpcHost,
      'grpc_port': instance.grpcPort,
      'grpc_server_cert': instance.grpcServerCert,
      'insecure': instance.insecure,
      'offline': instance.offline,
      'player_id': instance.playerId,
      'log_file': instance.logFile,
      'debug_level': instance.debugLevel,
    };

CreateDefaultConfig _$CreateDefaultConfigFromJson(Map<String, dynamic> json) =>
    CreateDefaultConfig(
      json['datadir'] as String,
      json['server_addr'] as String,
      json['grpc_cert_path'] as String,
      json['debug_level'] as String,
      json['br_rpc_url'] as String,
      json['br_client_cert'] as String,
      json['br_client_rpc_cert'] as String,
      json['br_client_rpc_key'] as String,
      json['rpc_user'] as String,
      json['rpc_pass'] as String,
    );

Map<String, dynamic> _$CreateDefaultConfigToJson(
        CreateDefaultConfig instance) =>
    <String, dynamic>{
      'datadir': instance.dataDir,
      'server_addr': instance.serverAddr,
      'grpc_cert_path': instance.grpcCertPath,
      'debug_level': instance.debugLevel,
      'br_rpc_url': instance.brRpcUrl,
      'br_client_cert': instance.brClientCert,
      'br_client_rpc_cert': instance.brClientRpcCert,
      'br_client_rpc_key': instance.brClientRpcKey,
      'rpc_user': instance.rpcUser,
      'rpc_pass': instance.rpcPass,
    };

IDInit _$IDInitFromJson(Map<String, dynamic> json) => IDInit(
      json['id'] as String,
      json['nick'] as String,
    );

Map<String, dynamic> _$IDInitToJson(IDInit instance) => <String, dynamic>{
      'id': instance.uid,
      'nick': instance.nick,
    };

GetUserNickArgs _$GetUserNickArgsFromJson(Map<String, dynamic> json) =>
    GetUserNickArgs(
      json['uid'] as String,
    );

Map<String, dynamic> _$GetUserNickArgsToJson(GetUserNickArgs instance) =>
    <String, dynamic>{
      'uid': instance.uid,
    };

LocalPlayer _$LocalPlayerFromJson(Map<String, dynamic> json) => LocalPlayer(
      json['uid'] as String,
      json['nick'] as String?,
      (json['bet_amt'] as num).toInt(),
      ready: json['ready'] as bool? ?? false,
    );

Map<String, dynamic> _$LocalPlayerToJson(LocalPlayer instance) =>
    <String, dynamic>{
      'uid': instance.uid,
      'nick': instance.nick,
      'bet_amt': instance.betAmount,
      'ready': instance.ready,
    };

LocalWaitingRoom _$LocalWaitingRoomFromJson(Map<String, dynamic> json) =>
    LocalWaitingRoom(
      json['id'] as String,
      json['host_id'] as String,
      (json['bet_amt'] as num).toInt(),
      players: (json['players'] as List<dynamic>?)
              ?.map((e) => LocalPlayer.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );

Map<String, dynamic> _$LocalWaitingRoomToJson(LocalWaitingRoom instance) =>
    <String, dynamic>{
      'id': instance.id,
      'host_id': instance.host,
      'bet_amt': instance.betAmt,
      'players': instance.players.map((e) => e.toJson()).toList(),
    };

LocalInfo _$LocalInfoFromJson(Map<String, dynamic> json) => LocalInfo(
      json['id'] as String,
      json['nick'] as String,
    );

Map<String, dynamic> _$LocalInfoToJson(LocalInfo instance) => <String, dynamic>{
      'id': instance.id,
      'nick': instance.nick,
    };

ServerCert _$ServerCertFromJson(Map<String, dynamic> json) => ServerCert(
      json['inner_fingerprint'] as String,
      json['outer_fingerprint'] as String,
    );

Map<String, dynamic> _$ServerCertToJson(ServerCert instance) =>
    <String, dynamic>{
      'inner_fingerprint': instance.innerFingerprint,
      'outer_fingerprint': instance.outerFingerprint,
    };

ServerInfo _$ServerInfoFromJson(Map<String, dynamic> json) => ServerInfo(
      innerFingerprint: json['innerFingerprint'] as String,
      outerFingerprint: json['outerFingerprint'] as String,
      serverAddr: json['serverAddr'] as String,
    );

Map<String, dynamic> _$ServerInfoToJson(ServerInfo instance) =>
    <String, dynamic>{
      'innerFingerprint': instance.innerFingerprint,
      'outerFingerprint': instance.outerFingerprint,
      'serverAddr': instance.serverAddr,
    };

RemoteUser _$RemoteUserFromJson(Map<String, dynamic> json) => RemoteUser(
      json['uid'] as String,
      json['nick'] as String,
    );

Map<String, dynamic> _$RemoteUserToJson(RemoteUser instance) =>
    <String, dynamic>{
      'uid': instance.uid,
      'nick': instance.nick,
    };

PublicIdentity _$PublicIdentityFromJson(Map<String, dynamic> json) =>
    PublicIdentity(
      json['name'] as String,
      json['nick'] as String,
      json['identity'] as String,
    );

Map<String, dynamic> _$PublicIdentityToJson(PublicIdentity instance) =>
    <String, dynamic>{
      'name': instance.name,
      'nick': instance.nick,
      'identity': instance.identity,
    };

Account _$AccountFromJson(Map<String, dynamic> json) => Account(
      json['name'] as String,
      (json['unconfirmed_balance'] as num).toInt(),
      (json['confirmed_balance'] as num).toInt(),
      (json['internal_key_count'] as num).toInt(),
      (json['external_key_count'] as num).toInt(),
    );

Map<String, dynamic> _$AccountToJson(Account instance) => <String, dynamic>{
      'name': instance.name,
      'unconfirmed_balance': instance.unconfirmedBalance,
      'confirmed_balance': instance.confirmedBalance,
      'internal_key_count': instance.internalKeyCount,
      'external_key_count': instance.externalKeyCount,
    };

LogEntry _$LogEntryFromJson(Map<String, dynamic> json) => LogEntry(
      json['from'] as String,
      json['message'] as String,
      json['internal'] as bool,
      (json['timestamp'] as num).toInt(),
    );

Map<String, dynamic> _$LogEntryToJson(LogEntry instance) => <String, dynamic>{
      'from': instance.from,
      'message': instance.message,
      'internal': instance.internal,
      'timestamp': instance.timestamp,
    };

SendOnChain _$SendOnChainFromJson(Map<String, dynamic> json) => SendOnChain(
      json['addr'] as String,
      (json['amount'] as num).toInt(),
      json['from_account'] as String,
    );

Map<String, dynamic> _$SendOnChainToJson(SendOnChain instance) =>
    <String, dynamic>{
      'addr': instance.addr,
      'amount': instance.amount,
      'from_account': instance.fromAccount,
    };

LoadUserHistory _$LoadUserHistoryFromJson(Map<String, dynamic> json) =>
    LoadUserHistory(
      json['uid'] as String,
      json['is_gc'] as bool,
      (json['page'] as num).toInt(),
      (json['page_num'] as num).toInt(),
    );

Map<String, dynamic> _$LoadUserHistoryToJson(LoadUserHistory instance) =>
    <String, dynamic>{
      'uid': instance.uid,
      'is_gc': instance.isGC,
      'page': instance.page,
      'page_num': instance.pageNum,
    };

WriteInvite _$WriteInviteFromJson(Map<String, dynamic> json) => WriteInvite(
      (json['fund_amount'] as num).toInt(),
      json['fund_account'] as String,
      json['gc_id'] as String?,
      json['prepaid'] as bool,
    );

Map<String, dynamic> _$WriteInviteToJson(WriteInvite instance) =>
    <String, dynamic>{
      'fund_amount': instance.fundAmount,
      'fund_account': instance.fundAccount,
      'gc_id': instance.gcid,
      'prepaid': instance.prepaid,
    };

RedeemedInviteFunds _$RedeemedInviteFundsFromJson(Map<String, dynamic> json) =>
    RedeemedInviteFunds(
      json['txid'] as String,
      (json['total'] as num).toInt(),
    );

Map<String, dynamic> _$RedeemedInviteFundsToJson(
        RedeemedInviteFunds instance) =>
    <String, dynamic>{
      'txid': instance.txid,
      'total': instance.total,
    };

CreateWaitingRoomArgs _$CreateWaitingRoomArgsFromJson(
        Map<String, dynamic> json) =>
    CreateWaitingRoomArgs(
      json['client_id'] as String,
      (json['bet_amt'] as num).toInt(),
      escrowId: json['escrow_id'] as String?,
    );

Map<String, dynamic> _$CreateWaitingRoomArgsToJson(
        CreateWaitingRoomArgs instance) =>
    <String, dynamic>{
      'client_id': instance.clientId,
      'bet_amt': instance.betAmt,
      'escrow_id': instance.escrowId,
    };

PokerTable _$PokerTableFromJson(Map<String, dynamic> json) => PokerTable(
      json['id'] as String,
      json['host_id'] as String,
      (json['small_blind'] as num).toInt(),
      (json['big_blind'] as num).toInt(),
      (json['max_players'] as num).toInt(),
      (json['min_players'] as num).toInt(),
      (json['current_players'] as num).toInt(),
      (json['min_balance'] as num).toInt(),
      (json['buy_in'] as num).toInt(),
      json['game_started'] as bool,
      json['all_players_ready'] as bool,
    );

Map<String, dynamic> _$PokerTableToJson(PokerTable instance) =>
    <String, dynamic>{
      'id': instance.id,
      'host_id': instance.hostId,
      'small_blind': instance.smallBlind,
      'big_blind': instance.bigBlind,
      'max_players': instance.maxPlayers,
      'min_players': instance.minPlayers,
      'current_players': instance.currentPlayers,
      'min_balance': instance.minBalance,
      'buy_in': instance.buyIn,
      'game_started': instance.gameStarted,
      'all_players_ready': instance.allPlayersReady,
    };

CreatePokerTableArgs _$CreatePokerTableArgsFromJson(
        Map<String, dynamic> json) =>
    CreatePokerTableArgs(
      (json['small_blind'] as num).toInt(),
      (json['big_blind'] as num).toInt(),
      (json['max_players'] as num).toInt(),
      (json['min_players'] as num).toInt(),
      (json['min_balance'] as num).toInt(),
      (json['buy_in'] as num).toInt(),
      (json['starting_chips'] as num).toInt(),
      (json['time_bank_seconds'] as num).toInt(),
      (json['auto_start_ms'] as num).toInt(),
    );

Map<String, dynamic> _$CreatePokerTableArgsToJson(
        CreatePokerTableArgs instance) =>
    <String, dynamic>{
      'small_blind': instance.smallBlind,
      'big_blind': instance.bigBlind,
      'max_players': instance.maxPlayers,
      'min_players': instance.minPlayers,
      'min_balance': instance.minBalance,
      'buy_in': instance.buyIn,
      'starting_chips': instance.startingChips,
      'time_bank_seconds': instance.timeBankSeconds,
      'auto_start_ms': instance.autoStartMs,
    };

JoinPokerTableArgs _$JoinPokerTableArgsFromJson(Map<String, dynamic> json) =>
    JoinPokerTableArgs(
      json['table_id'] as String,
    );

Map<String, dynamic> _$JoinPokerTableArgsToJson(JoinPokerTableArgs instance) =>
    <String, dynamic>{
      'table_id': instance.tableId,
    };

RunState _$RunStateFromJson(Map<String, dynamic> json) => RunState(
      dcrlndRunning: json['dcrlnd_running'] as bool,
      clientRunning: json['client_running'] as bool,
    );

Map<String, dynamic> _$RunStateToJson(RunState instance) => <String, dynamic>{
      'dcrlnd_running': instance.dcrlndRunning,
      'client_running': instance.clientRunning,
    };

ZipLogsArgs _$ZipLogsArgsFromJson(Map<String, dynamic> json) => ZipLogsArgs(
      json['include_golib'] as bool,
      json['include_ln'] as bool,
      json['only_last_file'] as bool,
      json['dest_path'] as String,
    );

Map<String, dynamic> _$ZipLogsArgsToJson(ZipLogsArgs instance) =>
    <String, dynamic>{
      'include_golib': instance.includeGolib,
      'include_ln': instance.includeLn,
      'only_last_file': instance.onlyLastFile,
      'dest_path': instance.destPath,
    };

UINotification _$UINotificationFromJson(Map<String, dynamic> json) =>
    UINotification(
      json['type'] as String,
      json['text'] as String,
      (json['count'] as num).toInt(),
      json['from'] as String,
    );

Map<String, dynamic> _$UINotificationToJson(UINotification instance) =>
    <String, dynamic>{
      'type': instance.type,
      'text': instance.text,
      'count': instance.count,
      'from': instance.from,
    };

UINotificationsConfig _$UINotificationsConfigFromJson(
        Map<String, dynamic> json) =>
    UINotificationsConfig(
      json['pms'] as bool,
      json['gcms'] as bool,
      json['gcmentions'] as bool,
    );

Map<String, dynamic> _$UINotificationsConfigToJson(
        UINotificationsConfig instance) =>
    <String, dynamic>{
      'pms': instance.pms,
      'gcms': instance.gcms,
      'gcmentions': instance.gcMentions,
    };
