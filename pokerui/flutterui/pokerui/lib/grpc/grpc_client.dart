import 'dart:io';
import 'package:fixnum/fixnum.dart' as fixnum;
import 'package:golib_plugin/grpc/generated/poker.pbgrpc.dart';
import 'package:grpc/grpc.dart';

class GrpcPokerClient {
  late ClientChannel _channel;
  late PokerServiceClient _pokerClient;
  late LobbyServiceClient _lobbyClient;

  GrpcPokerClient(String serverAddress, int port, {String? tlsCertPath}) {
    // Set up credentials based on whether TLS is being used
    final credentials = (tlsCertPath != null && tlsCertPath.isNotEmpty)
        ? _createSecureCredentials(tlsCertPath)
        : const ChannelCredentials.insecure();

    // Initialize the gRPC channel and client stubs
    _channel = ClientChannel(
      serverAddress,
      port: port,
      options: ChannelOptions(
        credentials: credentials,
      ),
    );
    _pokerClient = PokerServiceClient(_channel);
    _lobbyClient = LobbyServiceClient(_channel);
  }

  // Helper method to create secure credentials
  ChannelCredentials _createSecureCredentials(String certPath) {
    try {
      final cert = File(certPath).readAsBytesSync();
      return ChannelCredentials.secure(
        certificates: cert,
        authority: null, // Add authority if required
      );
    } catch (e) {
      throw Exception('Failed to read TLS certificate: $e');
    }
  }

  // ===== POKER SERVICE METHODS =====
  
  // Start game stream for real-time updates
  Stream<GameUpdate> startGameStream(String playerId, String tableId) async* {
    final request = StartGameStreamRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final responseStream = _pokerClient.startGameStream(request);
      await for (var response in responseStream) {
        yield response;
      }
    } catch (e) {
      print('Error during StartGameStream: $e');
      rethrow;
    }
  }

  // Show cards
  Future<ShowCardsResponse> showCards(String playerId, String tableId) async {
    final request = ShowCardsRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _pokerClient.showCards(request);
      return response;
    } catch (e) {
      print('Error during ShowCards: $e');
      rethrow;
    }
  }

  // Hide cards
  Future<HideCardsResponse> hideCards(String playerId, String tableId) async {
    final request = HideCardsRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _pokerClient.hideCards(request);
      return response;
    } catch (e) {
      print('Error during HideCards: $e');
      rethrow;
    }
  }

  // Make a bet
  Future<MakeBetResponse> makeBet(String playerId, String tableId, int amount) async {
    final request = MakeBetRequest()
      ..playerId = playerId
      ..tableId = tableId
      ..amount = fixnum.Int64(amount);

    try {
      final response = await _pokerClient.makeBet(request);
      return response;
    } catch (e) {
      print('Error during MakeBet: $e');
      rethrow;
    }
  }

  // Call a bet
  Future<CallBetResponse> callBet(String playerId, String tableId) async {
    final request = CallBetRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _pokerClient.callBet(request);
      return response;
    } catch (e) {
      print('Error during CallBet: $e');
      rethrow;
    }
  }

  // Fold a bet
  Future<FoldBetResponse> foldBet(String playerId, String tableId) async {
    final request = FoldBetRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _pokerClient.foldBet(request);
      return response;
    } catch (e) {
      print('Error during FoldBet: $e');
      rethrow;
    }
  }

  // Check a bet
  Future<CheckBetResponse> checkBet(String playerId, String tableId) async {
    final request = CheckBetRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _pokerClient.checkBet(request);
      return response;
    } catch (e) {
      print('Error during CheckBet: $e');
      rethrow;
    }
  }

  // Get current game state
  Future<GetGameStateResponse> getGameState(String tableId) async {
    final request = GetGameStateRequest()..tableId = tableId;

    try {
      final response = await _pokerClient.getGameState(request);
      return response;
    } catch (e) {
      print('Error during GetGameState: $e');
      rethrow;
    }
  }

  // Evaluate a hand
  Future<EvaluateHandResponse> evaluateHand(List<Card> cards) async {
    final request = EvaluateHandRequest()..cards.addAll(cards);

    try {
      final response = await _pokerClient.evaluateHand(request);
      return response;
    } catch (e) {
      print('Error during EvaluateHand: $e');
      rethrow;
    }
  }

  // Get last winners
  Future<GetLastWinnersResponse> getLastWinners(String tableId) async {
    final request = GetLastWinnersRequest()..tableId = tableId;

    try {
      final response = await _pokerClient.getLastWinners(request);
      return response;
    } catch (e) {
      print('Error during GetLastWinners: $e');
      rethrow;
    }
  }

  // ===== LOBBY SERVICE METHODS =====

  // Start notification stream
  Stream<Notification> startNotificationStream(String playerId) async* {
    final request = StartNotificationStreamRequest()..playerId = playerId;

    try {
      final responseStream = _lobbyClient.startNotificationStream(request);
      await for (var response in responseStream) {
        yield response;
      }
    } catch (e) {
      print('Error during StartNotificationStream: $e');
      rethrow;
    }
  }

  // Create a table
  Future<CreateTableResponse> createTable({
    required String playerId,
    required int smallBlind,
    required int bigBlind,
    required int maxPlayers,
    required int minPlayers,
    required int minBalance,
    required int buyIn,
    required int startingChips,
    int timeBankSeconds = 30,
    int autoStartMs = 0,
  }) async {
    final request = CreateTableRequest()
      ..playerId = playerId
      ..smallBlind = fixnum.Int64(smallBlind)
      ..bigBlind = fixnum.Int64(bigBlind)
      ..maxPlayers = maxPlayers
      ..minPlayers = minPlayers
      ..minBalance = fixnum.Int64(minBalance)
      ..buyIn = fixnum.Int64(buyIn)
      ..startingChips = fixnum.Int64(startingChips)
      ..timeBankSeconds = timeBankSeconds
      ..autoStartMs = autoStartMs;

    try {
      final response = await _lobbyClient.createTable(request);
      return response;
    } catch (e) {
      print('Error during CreateTable: $e');
      rethrow;
    }
  }

  // Join a table
  Future<JoinTableResponse> joinTable(String playerId, String tableId) async {
    final request = JoinTableRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _lobbyClient.joinTable(request);
      return response;
    } catch (e) {
      print('Error during JoinTable: $e');
      rethrow;
    }
  }

  // Leave a table
  Future<LeaveTableResponse> leaveTable(String playerId, String tableId) async {
    final request = LeaveTableRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _lobbyClient.leaveTable(request);
      return response;
    } catch (e) {
      print('Error during LeaveTable: $e');
      rethrow;
    }
  }

  // Get all tables
  Future<GetTablesResponse> getTables() async {
    final request = GetTablesRequest();

    try {
      final response = await _lobbyClient.getTables(request);
      return response;
    } catch (e) {
      print('Error during GetTables: $e');
      rethrow;
    }
  }

  // Get player's current table
  Future<GetPlayerCurrentTableResponse> getPlayerCurrentTable(String playerId) async {
    final request = GetPlayerCurrentTableRequest()..playerId = playerId;

    try {
      final response = await _lobbyClient.getPlayerCurrentTable(request);
      return response;
    } catch (e) {
      print('Error during GetPlayerCurrentTable: $e');
      rethrow;
    }
  }

  // Get player balance
  Future<GetBalanceResponse> getBalance(String playerId) async {
    final request = GetBalanceRequest()..playerId = playerId;

    try {
      final response = await _lobbyClient.getBalance(request);
      return response;
    } catch (e) {
      print('Error during GetBalance: $e');
      rethrow;
    }
  }

  // Update player balance
  Future<UpdateBalanceResponse> updateBalance(String playerId, int amount, String description) async {
    final request = UpdateBalanceRequest()
      ..playerId = playerId
      ..amount = fixnum.Int64(amount)
      ..description = description;

    try {
      final response = await _lobbyClient.updateBalance(request);
      return response;
    } catch (e) {
      print('Error during UpdateBalance: $e');
      rethrow;
    }
  }

  // Process a tip
  Future<ProcessTipResponse> processTip(String fromPlayerId, String toPlayerId, int amount, String message) async {
    final request = ProcessTipRequest()
      ..fromPlayerId = fromPlayerId
      ..toPlayerId = toPlayerId
      ..amount = fixnum.Int64(amount)
      ..message = message;

    try {
      final response = await _lobbyClient.processTip(request);
      return response;
    } catch (e) {
      print('Error during ProcessTip: $e');
      rethrow;
    }
  }

  // Set player ready
  Future<SetPlayerReadyResponse> setPlayerReady(String playerId, String tableId) async {
    final request = SetPlayerReadyRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _lobbyClient.setPlayerReady(request);
      return response;
    } catch (e) {
      print('Error during SetPlayerReady: $e');
      rethrow;
    }
  }

  // Set player unready
  Future<SetPlayerUnreadyResponse> setPlayerUnready(String playerId, String tableId) async {
    final request = SetPlayerUnreadyRequest()
      ..playerId = playerId
      ..tableId = tableId;

    try {
      final response = await _lobbyClient.setPlayerUnready(request);
      return response;
    } catch (e) {
      print('Error during SetPlayerUnready: $e');
      rethrow;
    }
  }

  // Optionally, clean up the gRPC connection
  Future<void> shutdown() async {
    await _channel.shutdown();
  }
}
