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
  json['sounds_enabled'] as bool,
);

Map<String, dynamic> _$InitClientToJson(InitClient instance) =>
    <String, dynamic>{
      'server_addr': instance.serverAddr,
      'grpc_cert_path': instance.grpcCertPath,
      'datadir': instance.dataDir,
      'payout_address': instance.payoutAddress,
      'log_file': instance.logFile,
      'debug_level': instance.debugLevel,
      'sounds_enabled': instance.soundsEnabled,
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
    );

Map<String, dynamic> _$CreateDefaultConfigToJson(
  CreateDefaultConfig instance,
) => <String, dynamic>{
  'datadir': instance.dataDir,
  'server_addr': instance.serverAddr,
  'grpc_cert_path': instance.grpcCertPath,
  'debug_level': instance.debugLevel,
};

UpdateConfig _$UpdateConfigFromJson(Map<String, dynamic> json) => UpdateConfig(
  json['datadir'] as String,
  json['server_addr'] as String,
  json['grpc_cert_path'] as String,
  json['address'] as String,
  json['debug_level'] as String,
  json['table_theme'] as String,
  json['card_theme'] as String,
  json['card_size'] as String,
  json['ui_size'] as String,
  json['sounds_enabled'] as bool,
  json['hide_table_logo'] as bool,
  json['logo_position'] as String,
);

Map<String, dynamic> _$UpdateConfigToJson(UpdateConfig instance) =>
    <String, dynamic>{
      'datadir': instance.dataDir,
      'server_addr': instance.serverAddr,
      'grpc_cert_path': instance.grpcCertPath,
      'address': instance.address,
      'debug_level': instance.debugLevel,
      'table_theme': instance.tableTheme,
      'card_theme': instance.cardTheme,
      'card_size': instance.cardSize,
      'ui_size': instance.uiSize,
      'sounds_enabled': instance.soundsEnabled,
      'hide_table_logo': instance.hideTableLogo,
      'logo_position': instance.logoPosition,
    };

IDInit _$IDInitFromJson(Map<String, dynamic> json) =>
    IDInit(json['id'] as String, json['nick'] as String);

Map<String, dynamic> _$IDInitToJson(IDInit instance) => <String, dynamic>{
  'id': instance.uid,
  'nick': instance.nick,
};

GetUserNickArgs _$GetUserNickArgsFromJson(Map<String, dynamic> json) =>
    GetUserNickArgs(json['uid'] as String);

Map<String, dynamic> _$GetUserNickArgsToJson(GetUserNickArgs instance) =>
    <String, dynamic>{'uid': instance.uid};

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
      players:
          (json['players'] as List<dynamic>?)
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

LocalInfo _$LocalInfoFromJson(Map<String, dynamic> json) =>
    LocalInfo(json['id'] as String, json['nick'] as String);

Map<String, dynamic> _$LocalInfoToJson(LocalInfo instance) => <String, dynamic>{
  'id': instance.id,
  'nick': instance.nick,
};

RegisterRequest _$RegisterRequestFromJson(Map<String, dynamic> json) =>
    RegisterRequest(json['nickname'] as String);

Map<String, dynamic> _$RegisterRequestToJson(RegisterRequest instance) =>
    <String, dynamic>{'nickname': instance.nickname};

LoginRequest _$LoginRequestFromJson(Map<String, dynamic> json) =>
    LoginRequest(json['nickname'] as String);

Map<String, dynamic> _$LoginRequestToJson(LoginRequest instance) =>
    <String, dynamic>{'nickname': instance.nickname};

LoginResponse _$LoginResponseFromJson(Map<String, dynamic> json) =>
    LoginResponse(
      json['token'] as String,
      json['user_id'] as String,
      json['nickname'] as String,
      json['address'] as String,
    );

Map<String, dynamic> _$LoginResponseToJson(LoginResponse instance) =>
    <String, dynamic>{
      'token': instance.token,
      'user_id': instance.userId,
      'nickname': instance.nickname,
      'address': instance.address,
    };

RequestLoginCodeResponse _$RequestLoginCodeResponseFromJson(
  Map<String, dynamic> json,
) => RequestLoginCodeResponse(
  json['code'] as String,
  (json['ttl_sec'] as num).toInt(),
  json['address_hint'] as String,
);

Map<String, dynamic> _$RequestLoginCodeResponseToJson(
  RequestLoginCodeResponse instance,
) => <String, dynamic>{
  'code': instance.code,
  'ttl_sec': instance.ttlSec,
  'address_hint': instance.addressHint,
};

SetPayoutAddressRequest _$SetPayoutAddressRequestFromJson(
  Map<String, dynamic> json,
) => SetPayoutAddressRequest(
  json['address'] as String,
  json['signature'] as String,
  json['code'] as String,
);

Map<String, dynamic> _$SetPayoutAddressRequestToJson(
  SetPayoutAddressRequest instance,
) => <String, dynamic>{
  'address': instance.address,
  'signature': instance.signature,
  'code': instance.code,
};

SetPayoutAddressResponse _$SetPayoutAddressResponseFromJson(
  Map<String, dynamic> json,
) => SetPayoutAddressResponse(
  json['ok'] as bool? ?? false,
  json['error'] as String? ?? '',
  json['address'] as String? ?? '',
);

Map<String, dynamic> _$SetPayoutAddressResponseToJson(
  SetPayoutAddressResponse instance,
) => <String, dynamic>{
  'ok': instance.ok,
  'error': instance.error,
  'address': instance.address,
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

RemoteUser _$RemoteUserFromJson(Map<String, dynamic> json) =>
    RemoteUser(json['uid'] as String, json['nick'] as String);

Map<String, dynamic> _$RemoteUserToJson(RemoteUser instance) =>
    <String, dynamic>{'uid': instance.uid, 'nick': instance.nick};

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
    RedeemedInviteFunds(json['txid'] as String, (json['total'] as num).toInt());

Map<String, dynamic> _$RedeemedInviteFundsToJson(
  RedeemedInviteFunds instance,
) => <String, dynamic>{'txid': instance.txid, 'total': instance.total};

CreateWaitingRoomArgs _$CreateWaitingRoomArgsFromJson(
  Map<String, dynamic> json,
) => CreateWaitingRoomArgs(
  json['client_id'] as String,
  (json['bet_amt'] as num).toInt(),
  escrowId: json['escrow_id'] as String?,
);

Map<String, dynamic> _$CreateWaitingRoomArgsToJson(
  CreateWaitingRoomArgs instance,
) => <String, dynamic>{
  'client_id': instance.clientId,
  'bet_amt': instance.betAmt,
  'escrow_id': instance.escrowId,
};

PokerTable _$PokerTableFromJson(Map<String, dynamic> json) => PokerTable(
  json['id'] as String,
  (json['small_blind'] as num).toInt(),
  (json['big_blind'] as num).toInt(),
  (json['max_players'] as num).toInt(),
  (json['min_players'] as num).toInt(),
  (json['current_players'] as num).toInt(),
  (json['buy_in'] as num).toInt(),
  json['game_started'] as bool,
  json['all_players_ready'] as bool,
  players: (json['players'] as List<dynamic>?)
      ?.map((e) => PlayerDTO.fromJson(e as Map<String, dynamic>))
      .toList(),
  blindIncreaseIntervalSec:
      (json['blind_increase_interval_sec'] as num?)?.toInt() ?? 0,
);

Map<String, dynamic> _$PokerTableToJson(PokerTable instance) =>
    <String, dynamic>{
      'id': instance.id,
      'small_blind': instance.smallBlind,
      'big_blind': instance.bigBlind,
      'max_players': instance.maxPlayers,
      'min_players': instance.minPlayers,
      'current_players': instance.currentPlayers,
      'buy_in': instance.buyIn,
      'game_started': instance.gameStarted,
      'all_players_ready': instance.allPlayersReady,
      'players': instance.players,
      'blind_increase_interval_sec': instance.blindIncreaseIntervalSec,
    };

CreatePokerTableArgs _$CreatePokerTableArgsFromJson(
  Map<String, dynamic> json,
) => CreatePokerTableArgs(
  (json['small_blind'] as num).toInt(),
  (json['big_blind'] as num).toInt(),
  (json['max_players'] as num).toInt(),
  (json['min_players'] as num).toInt(),
  (json['buy_in'] as num).toInt(),
  (json['starting_chips'] as num).toInt(),
  (json['time_bank_seconds'] as num).toInt(),
  (json['auto_start_ms'] as num).toInt(),
  (json['auto_advance_ms'] as num).toInt(),
  blindIncreaseIntervalSec:
      (json['blind_increase_interval_sec'] as num?)?.toInt() ?? 0,
);

Map<String, dynamic> _$CreatePokerTableArgsToJson(
  CreatePokerTableArgs instance,
) => <String, dynamic>{
  'small_blind': instance.smallBlind,
  'big_blind': instance.bigBlind,
  'max_players': instance.maxPlayers,
  'min_players': instance.minPlayers,
  'buy_in': instance.buyIn,
  'starting_chips': instance.startingChips,
  'time_bank_seconds': instance.timeBankSeconds,
  'auto_start_ms': instance.autoStartMs,
  'auto_advance_ms': instance.autoAdvanceMs,
  'blind_increase_interval_sec': instance.blindIncreaseIntervalSec,
};

MakeBetArgs _$MakeBetArgsFromJson(Map<String, dynamic> json) =>
    MakeBetArgs((json['amount'] as num).toInt());

Map<String, dynamic> _$MakeBetArgsToJson(MakeBetArgs instance) =>
    <String, dynamic>{'amount': instance.amount};

EvaluateHandArgs _$EvaluateHandArgsFromJson(Map<String, dynamic> json) =>
    EvaluateHandArgs(
      (json['cards'] as List<dynamic>)
          .map((e) => CardArg.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$EvaluateHandArgsToJson(EvaluateHandArgs instance) =>
    <String, dynamic>{'cards': instance.cards};

CardArg _$CardArgFromJson(Map<String, dynamic> json) =>
    CardArg((json['suit'] as num).toInt(), (json['value'] as num).toInt());

Map<String, dynamic> _$CardArgToJson(CardArg instance) => <String, dynamic>{
  'suit': instance.suit,
  'value': instance.value,
};

JoinPokerTableArgs _$JoinPokerTableArgsFromJson(Map<String, dynamic> json) =>
    JoinPokerTableArgs(json['table_id'] as String);

Map<String, dynamic> _$JoinPokerTableArgsToJson(JoinPokerTableArgs instance) =>
    <String, dynamic>{'table_id': instance.tableId};

CardDTO _$CardDTOFromJson(Map<String, dynamic> json) =>
    CardDTO(json['suit'] as String, json['value'] as String);

Map<String, dynamic> _$CardDTOToJson(CardDTO instance) => <String, dynamic>{
  'suit': instance.suit,
  'value': instance.value,
};

PlayerDTO _$PlayerDTOFromJson(Map<String, dynamic> json) => PlayerDTO(
  json['id'] as String,
  json['name'] as String,
  (json['balance'] as num).toInt(),
  (json['hand'] as List<dynamic>)
      .map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
      .toList(),
  (json['currentBet'] as num).toInt(),
  json['folded'] as bool,
  json['isTurn'] as bool,
  json['isAllIn'] as bool,
  json['isDealer'] as bool,
  json['isReady'] as bool,
  json['disconnected'] as bool,
  json['handDescription'] as String,
  (json['playerState'] as num).toInt(),
  json['isSmallBlind'] as bool,
  json['isBigBlind'] as bool,
  json['escrowId'] as String? ?? '',
  json['escrowReady'] as bool? ?? false,
  (json['tableSeat'] as num?)?.toInt() ?? 0,
  json['cardsRevealed'] as bool? ?? false,
);

Map<String, dynamic> _$PlayerDTOToJson(PlayerDTO instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'balance': instance.balance,
  'hand': instance.hand,
  'currentBet': instance.currentBet,
  'folded': instance.folded,
  'isTurn': instance.isTurn,
  'isAllIn': instance.isAllIn,
  'isDealer': instance.isDealer,
  'isReady': instance.isReady,
  'disconnected': instance.disconnected,
  'handDescription': instance.handDescription,
  'playerState': instance.playerState,
  'isSmallBlind': instance.isSmallBlind,
  'isBigBlind': instance.isBigBlind,
  'escrowId': instance.escrowId,
  'escrowReady': instance.escrowReady,
  'tableSeat': instance.tableSeat,
  'cardsRevealed': instance.cardsRevealed,
};

GameUpdateDTO _$GameUpdateDTOFromJson(Map<String, dynamic> json) =>
    GameUpdateDTO(
      json['tableId'] as String,
      (json['phase'] as num).toInt(),
      (json['players'] as List<dynamic>)
          .map((e) => PlayerDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      (json['communityCards'] as List<dynamic>)
          .map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      (json['pot'] as num).toInt(),
      (json['currentBet'] as num).toInt(),
      json['currentPlayer'] as String,
      (json['minRaise'] as num).toInt(),
      (json['maxRaise'] as num).toInt(),
      json['gameStarted'] as bool,
      (json['playersRequired'] as num).toInt(),
      (json['playersJoined'] as num).toInt(),
      json['phaseName'] as String,
      (json['timeBankSeconds'] as num).toInt(),
      (json['turnDeadlineUnixMs'] as num).toInt(),
      (json['smallBlind'] as num?)?.toInt() ?? 0,
      (json['bigBlind'] as num?)?.toInt() ?? 0,
      blindLevel: (json['blindLevel'] as num?)?.toInt() ?? 0,
      nextBlindIncreaseUnixMs:
          (json['nextBlindIncreaseUnixMs'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$GameUpdateDTOToJson(GameUpdateDTO instance) =>
    <String, dynamic>{
      'tableId': instance.tableId,
      'phase': instance.phase,
      'players': instance.players,
      'communityCards': instance.communityCards,
      'pot': instance.pot,
      'currentBet': instance.currentBet,
      'currentPlayer': instance.currentPlayer,
      'minRaise': instance.minRaise,
      'maxRaise': instance.maxRaise,
      'gameStarted': instance.gameStarted,
      'playersRequired': instance.playersRequired,
      'playersJoined': instance.playersJoined,
      'phaseName': instance.phaseName,
      'timeBankSeconds': instance.timeBankSeconds,
      'turnDeadlineUnixMs': instance.turnDeadlineUnixMs,
      'smallBlind': instance.smallBlind,
      'bigBlind': instance.bigBlind,
      'blindLevel': instance.blindLevel,
      'nextBlindIncreaseUnixMs': instance.nextBlindIncreaseUnixMs,
    };

WinnerDTO _$WinnerDTOFromJson(Map<String, dynamic> json) => WinnerDTO(
  json['playerId'] as String,
  (json['handRank'] as num).toInt(),
  (json['winnings'] as num).toInt(),
  bestHand: (json['bestHand'] as List<dynamic>?)
      ?.map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$WinnerDTOToJson(WinnerDTO instance) => <String, dynamic>{
  'playerId': instance.playerId,
  'handRank': instance.handRank,
  'bestHand': instance.bestHand,
  'winnings': instance.winnings,
};

ShowdownPlayerDTO _$ShowdownPlayerDTOFromJson(Map<String, dynamic> json) =>
    ShowdownPlayerDTO(
      json['playerId'] as String,
      holeCards: (json['holeCards'] as List<dynamic>?)
          ?.map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      finalState: (json['finalState'] as num?)?.toInt(),
      handRank: (json['handRank'] as num?)?.toInt(),
      bestHand: (json['bestHand'] as List<dynamic>?)
          ?.map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      contribution: (json['contribution'] as num?)?.toInt(),
    );

Map<String, dynamic> _$ShowdownPlayerDTOToJson(ShowdownPlayerDTO instance) =>
    <String, dynamic>{
      'playerId': instance.playerId,
      'holeCards': instance.holeCards,
      'finalState': instance.finalState,
      'handRank': instance.handRank,
      'bestHand': instance.bestHand,
      'contribution': instance.contribution,
    };

NotificationDTO _$NotificationDTOFromJson(Map<String, dynamic> json) =>
    NotificationDTO(
      (json['type'] as num).toInt(),
      message: json['message'] as String?,
      tableId: json['tableId'] as String?,
      playerId: json['playerId'] as String?,
      cards: (json['cards'] as List<dynamic>?)
          ?.map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      amount: (json['amount'] as num?)?.toInt(),
      newBalance: (json['newBalance'] as num?)?.toInt(),
      ready: json['ready'] as bool?,
      started: json['started'] as bool?,
      gameReadyToPlay: json['gameReadyToPlay'] as bool?,
      countdown: (json['countdown'] as num?)?.toInt(),
      table: json['table'] == null
          ? null
          : PokerTable.fromJson(json['table'] as Map<String, dynamic>),
      winners: (json['winners'] as List<dynamic>?)
          ?.map((e) => WinnerDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      showdownPot: (json['showdownPot'] as num?)?.toInt(),
      showdownPlayers: (json['players'] as List<dynamic>?)
          ?.map((e) => ShowdownPlayerDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
      board: (json['board'] as List<dynamic>?)
          ?.map((e) => CardDTO.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$NotificationDTOToJson(NotificationDTO instance) =>
    <String, dynamic>{
      'type': instance.type,
      'message': instance.message,
      'tableId': instance.tableId,
      'playerId': instance.playerId,
      'cards': instance.cards,
      'amount': instance.amount,
      'newBalance': instance.newBalance,
      'ready': instance.ready,
      'started': instance.started,
      'gameReadyToPlay': instance.gameReadyToPlay,
      'countdown': instance.countdown,
      'table': instance.table,
      'winners': instance.winners,
      'showdownPot': instance.showdownPot,
      'players': instance.showdownPlayers,
      'board': instance.board,
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
  Map<String, dynamic> json,
) => UINotificationsConfig(
  json['pms'] as bool,
  json['gcms'] as bool,
  json['gcmentions'] as bool,
);

Map<String, dynamic> _$UINotificationsConfigToJson(
  UINotificationsConfig instance,
) => <String, dynamic>{
  'pms': instance.pms,
  'gcms': instance.gcms,
  'gcmentions': instance.gcMentions,
};
