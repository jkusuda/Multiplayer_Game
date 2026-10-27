// Calls the Go backend from Unity. Copy into Assets/Scripts/Architecture/.
// This is the only file in the game that knows the backend's URLs and JSON.
using System;
using System.Text;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Game.Backend
{
    // JsonUtility matches public field names to JSON keys exactly,
    // so these are named as the server names them.

    [Serializable]
    public class HealthReply
    {
        public string status;
    }

    [Serializable]
    public class CreatePlayerRequest
    {
        public string display_name;
    }

    [Serializable]
    public class PlayerReply
    {
        public long id;
        public string display_name;
        public string created_at;
    }

    [Serializable]
    public class CreateRoomRequest
    {
        public long host_player_id;
        public string relay_join_code;
        public int max_players;
    }

    [Serializable]
    public class RoomReply
    {
        public long id;
        public string code;
        public long host_player_id;
        public string status;
    }

    [Serializable]
    public class RoomPlayerRequest
    {
        public long player_id;
    }

    [Serializable]
    public class JoinRoomReply
    {
        public string code;
        public string relay_join_code;
        public long host_player_id;
    }

    [Serializable]
    public class BackendError
    {
        public string error;
        public string message;
    }

    // Thrown when a call fails. Code is the server's error code
    // (for example "room_full"), or "unreachable" if there was no answer.
    public class BackendException : Exception
    {
        public readonly long Status;
        public readonly string Code;

        public BackendException(long status, string code, string message) : base(message)
        {
            Status = status;
            Code = code;
        }
    }

    public class BackendClient
    {
        readonly string baseUrl;

        // BackendClient talks to the server at baseUrl, e.g. "http://localhost:8080".
        public BackendClient(string baseUrl)
        {
            this.baseUrl = baseUrl.TrimEnd('/');
        }

        // Ping checks the server and its database are up.
        public Task<HealthReply> Ping()
        {
            return Send<HealthReply>("GET", "/healthz", null);
        }

        // CreatePlayer adds a player; save the reply's id and reuse it.
        public Task<PlayerReply> CreatePlayer(string displayName)
        {
            return Send<PlayerReply>("POST", "/players", new CreatePlayerRequest { display_name = displayName });
        }

        // GetPlayer looks up one player by id.
        public Task<PlayerReply> GetPlayer(long playerId)
        {
            return Send<PlayerReply>("GET", "/players/" + playerId, null);
        }

        // CreateRoom opens a room for a Relay game the host already started,
        // and returns the room code to show other players.
        public Task<RoomReply> CreateRoom(long hostPlayerId, string relayJoinCode, int maxPlayers)
        {
            return Send<RoomReply>("POST", "/rooms", new CreateRoomRequest
            {
                host_player_id = hostPlayerId,
                relay_join_code = relayJoinCode,
                max_players = maxPlayers,
            });
        }

        // JoinRoom adds a player to a room and returns the Relay code to connect with.
        public Task<JoinRoomReply> JoinRoom(string roomCode, long playerId)
        {
            return Send<JoinRoomReply>("POST", RoomPath(roomCode) + "/join", new RoomPlayerRequest { player_id = playerId });
        }

        // LeaveRoom tells the server a player left. If the host leaves, the room closes.
        public Task LeaveRoom(string roomCode, long playerId)
        {
            return Send<object>("POST", RoomPath(roomCode) + "/leave", new RoomPlayerRequest { player_id = playerId });
        }

        // RoomPath builds /rooms/{code}, cleaning up however the player typed the code.
        static string RoomPath(string roomCode)
        {
            return "/rooms/" + Uri.EscapeDataString((roomCode ?? "").Trim().ToUpperInvariant());
        }

        // Send makes one request, with body as JSON if not null, and reads the reply into T.
        // It throws BackendException on an error reply or no reply.
        async Task<T> Send<T>(string method, string path, object body)
        {
            using (var request = new UnityWebRequest(baseUrl + path, method))
            {
                request.downloadHandler = new DownloadHandlerBuffer();
                request.timeout = 8; // seconds

                if (body != null)
                {
                    byte[] json = Encoding.UTF8.GetBytes(JsonUtility.ToJson(body));
                    request.uploadHandler = new UploadHandlerRaw(json);
                    request.SetRequestHeader("Content-Type", "application/json");
                }

                // Resumes on the main thread, so callers can touch UI afterwards.
                var finished = new TaskCompletionSource<bool>();
                request.SendWebRequest().completed += _ => finished.SetResult(true);
                await finished.Task;

                string text = request.downloadHandler.text;

                if (request.result == UnityWebRequest.Result.Success)
                {
                    // 204 replies (like LeaveRoom) have no body.
                    return string.IsNullOrEmpty(text) ? default : JsonUtility.FromJson<T>(text);
                }

                if (request.result == UnityWebRequest.Result.ProtocolError)
                {
                    var error = new BackendError();
                    try { JsonUtility.FromJsonOverwrite(text, error); } catch (ArgumentException) { }
                    throw new BackendException(request.responseCode,
                        string.IsNullOrEmpty(error.error) ? "http_" + request.responseCode : error.error,
                        string.IsNullOrEmpty(error.message) ? request.error : error.message);
                }

                throw new BackendException(0, "unreachable", request.error);
            }
        }
    }
}
