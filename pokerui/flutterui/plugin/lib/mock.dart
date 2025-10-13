import 'dart:async';
import 'dart:io';
// import 'dart:typed_data';

import 'package:flutter/cupertino.dart';
import 'package:flutter/services.dart';

import 'definitions.dart';
import 'package:path/path.dart' as path;
// import 'package:shelf_web_socket/shelf_web_socket.dart';
import 'package:json_rpc_2/json_rpc_2.dart';
// import 'package:web_socket_channel/web_socket_channel.dart';

// Throws the given exception or returns null. Use it as:
// _threw(e) ?? otherval
dynamic _threw(Exception? e) {
  if (e != null) throw e;
  return null;
}

Exception? _exception(Parameter p) =>
    p.asStringOr("") != "" ? Exception(p.asString) : null;

int _lastEID = 0;
int nextEID() {
  return _lastEID++;
}

class MockPlugin with NtfStreams /*implements PluginPlatform*/ {
  /// ******************************************
  /// Fields
  ///******************************************
  Directory tempRoot = Directory.systemTemp.createTempSync("fd-mock");
  String tag = "";
  StreamController<String> streamCtrl = StreamController<String>();
  Exception? failNextGetURL;
  Exception? failNextConnect;
  Map<String, List<String>> gcBooks = {
    "Test Group": ["user1", "user2", "user3"],
    "group2": ["bleh", "booo", "fran"],
  };

  /// ******************************************
  /// Constructor
  ///******************************************
  MockPlugin() {
    /*
    webSocketHandler((WebSocketChannel socket) {
      final server = Server(socket.cast<String>());

      server.registerMethod('hello', rpcHello);
      server.registerMethod('failNextGetURL', rpcFailNextGetURL);
      server.registerMethod('failNextConnect', rpcFailNextConnect);
      server.registerMethod('recvMsg', rpcRecvMsg);
      server.registerMethod('feedPost', rpcFeedPost);

      server.listen();
    });
    */

    /*
    () async {
      final address = "127.0.0.1";
      final port = 4042;
      await shelf_io.serve(handler, address, port);
      print("Mock ctrl server listening on ws://$address:$port");
    }();
    */

    // Send some initial feed events.
    /*
    () async {
      ntfPostsFeed.add(FeedPost(
          "Someone",
          "xxxxx",
          "My first content. This is a sample of the stuff I'll add in the future.",
          "test.md",
          DateTime.parse("2021-08-18 15:18:22")));
      ntfPostsFeed.add(FeedPost(
        "Someone else",
        "xxxxx",
        "This is someone else. Hope you're all fine.",
        "test.md",
        DateTime.parse("2021-08-18 15:18:22"),
      ));
      ntfPostsFeed.add(FeedPost(
        "Someone",
        "xxxxx",
        "My second content. Maybe later there will be more stuff.",
        "test.md",
        DateTime.parse("2021-08-18 15:18:22"),
      ));
      ntfPostsFeed.add(FeedPost(
        "Third Party11",
        "xxxxx",
        "Content from third party. This'll probably fail to be fetched.",
        "*bug",
        DateTime.parse("2021-08-18 15:18:22"),
      ));
    }();
    */
  }

  /// ******************************************
  ///  PluginPlatform implementation methods.
  ///******************************************

  Future<String?> get platformVersion async => "mock 1.0";
  String get majorPlatform => "mock";
  String get minorPlatform => "mock";
  Future<void> setTag(String t) async => tag = t;
  Future<void> hello() async => debugPrint("hello from mock");
  Future<String> getURL(String url) async =>
      _threw(failNextGetURL) ?? "xxx.xxx.xxx.xxx";

  Future<String> nextTime() async => "$tag ${DateTime.now().toIso8601String()}";
  Future<void> writeStr(String s) async => streamCtrl.add(s);
  Stream<String> readStream() => streamCtrl.stream;

  Future<String> asyncCall(int cmd, dynamic payload) async =>
      throw "unimplemented";

  Future<String> asyncHello(String name) async => throw "unimplemented";

  Future<void> initClient(InitClient args) async {}

  Future<bool> hasServer() async =>
      Future.delayed(const Duration(seconds: 3), () => false);

  Future<void> initID(IDInit args) async => throw "unimplemented";

  Future<void> replyConfServerCert(bool accept) async {}

  Future<String> userNick(String uid) async => throw "unimplemented";
  Future<void> commentPost(
          String from, String pid, String comment, String? parent) async =>
      throw "unimplemented";
  Future<LocalInfo> getLocalInfo() async => throw "unimplemented";
  Future<void> requestMediateID(String mediator, String target) async =>
      throw "unimplemented";
  Future<void> kxSearchPostAuthor(String from, String pid) async =>
      throw "unimplemented";
  Future<void> relayPostToAll(String from, String pid) async =>
      throw "unimplemented";
  Future<Map<String, dynamic>> getGCBlockList(String gcID) async =>
      throw "unimplemented";
  Future<void> addToGCBlockList(String gcID, String uid) async =>
      throw "unimplemented";
  Future<void> removeFromGCBlockList(String gcID, String uid) async =>
      throw "unimplemented";
  Future<void> partFromGC(String gcID) async => throw "unimplemented";
  Future<void> killGC(String gcID) async => throw "unimplemented";
  Future<void> blockUser(String uid) async => throw "unimplemented";
  Future<void> ignoreUser(String uid) async => throw "unimplemented";
  Future<void> unignoreUser(String uid) async => throw "unimplemented";
  Future<bool> isIgnored(String uid) async => throw "unimplemented";
  Future<List<String>> listSubscribers() async => throw "unimplemented";
  Future<List<String>> listSubscriptions() async => throw "unimplemented";
  Future<String> lnRunDcrlnd(String rootPath, String network, String password,
          String proxyaddr, bool torisolation) async =>
      throw "unimplemented";
  void captureDcrlndLog() => throw "unimplemented";
  Future<String> lnGetDepositAddr() async => throw "unimplemented";
  Future<void> lnRequestRecvCapacity(
          String server, String key, double chanSize) async =>
      throw "unimplemented";
  Future<void> lnConfirmPayReqRecvChan(bool value) async =>
      throw "unimplemented";
  Future<void> confirmFileDownload(String fid, bool confirm) async =>
      throw "unimplemented";
  Future<void> sendFile(String uid, String filepath) async =>
      throw "unimplemented";

  /// ******************************************
  ///  Mock-only Methods (to be added to PluginPlatform)
  ///******************************************

  Future<ServerInfo> connectToServer(
          String server, String name, String nick) async =>
      Future<ServerInfo>.delayed(
          const Duration(seconds: 3),
          () =>
              _threw(failNextConnect) ??
              ServerInfo(
                  innerFingerprint: "XXYY",
                  outerFingerprint: "LLOOOO",
                  serverAddr: server));




  Future<void> transitiveInvite(String destNick, String targetNick) async {
    throw "unimplemented";
    /*
    chatMsgsCtrl.add(ChatMsg.gc(
        nextEID(), destNick, destNick, "Sent invite for user $targetNick",
        isServerMsg: true));
    */
  }

  Future<void> requestKXReset(String uid) async => throw "unimplemented";

  Future<void> shareFile(
          String filename, String? uid, double cost, String descr) =>
      throw "unimplemented";

  Future<void> unshareFile(String filename, String? uid) =>
      throw "unimplemented";


  // extractMdContent extracts the given content (which must be a native bundle
  // format) and returns the dir to the extracted temp bundle.
  Future<String> extractMdContent(String nick, String filename) async =>
      Future.delayed(const Duration(seconds: 3), () async {
        if (filename == "*bug") {
          throw Exception("Bugging out as requested");
        }

        var dir = Directory(path.join(tempRoot.path, "sample_md"));
        if (dir.existsSync()) {
          return dir.path;
        }

        // Extract sample md data to it.
        dir.createSync(recursive: true);
        List<String> files = ["index.md", "bunny_small.mp4", "pixabay.jpg"];
        for (var fname in files) {
          File f = File(path.join(dir.path, fname));
          var content = await rootBundle.load("assets/sample_md/$fname");
          var buffer = content.buffer;
          var bytes =
              buffer.asUint8List(content.offsetInBytes, content.lengthInBytes);
          await f.writeAsBytes(bytes);
        }

        return dir.path;
      });

  Future<void> subscribeToPosts(String uid) => throw "unimplemented";

  /// ******************************************
  ///  Mock JSON-RPC handlers.
  ///******************************************
  String rpcHello() => "is it me you're looking for?";

  void rpcFailNextGetURL(Parameters params) {
    failNextGetURL = _exception(params[0]);
  }

  void rpcFailNextConnect(Parameters params) {
    failNextConnect = _exception(params[0]);
  }

  void rpcRecvMsg(Parameters params) {
    /*
    var nick = params[0].asString;
    var msg = params[1].asString;
    chatMsgsCtrl.add(ChatMsg.pm(nextEID(), nick, nick, msg, mine: false));
    */
    throw "unimplemented";
  }

  void rpcFeedPost(Parameters params) {
    /*
    ntfPostsFeed.add(PostSummary(params[0].asString, "xxxxx", params[1].asString,
        "test.md", DateTime.now()));
        */
    throw "unimplemented";
  }
}
