// Copy this file into the Unity project (for example Assets/_Game/Net/).
// It should stay the only place in the game that knows the backend's URLs and JSON.
//
// This file is plumbing: sending a request and reading the reply.
// Add one method per endpoint as you build them, following Ping() below.
using System;
using System.Text;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Game.Backend
{
    // ---- Data shapes ----
    // One [Serializable] class per request or reply. Unity's JsonUtility matches
    // public field names to JSON keys exactly, so name them as the server does.

    [Serializable]
    public class HealthReply
    {
        public string status;
    }

    [Serializable]
    public class BackendError
    {
        public string error;
        public string message;
    }

    /// <summary>Thrown when a call fails. Check Code to decide what to show the player.</summary>
    public class BackendException : Exception
    {
        public readonly long Status; // HTTP status, or 0 if the server could not be reached
        public readonly string Code; // the server's error code, or "unreachable"

        public BackendException(long status, string code, string message) : base(message)
        {
            Status = status;
            Code = code;
        }
    }

    public class BackendClient
    {
        readonly string baseUrl;

        public BackendClient(string baseUrl)
        {
            this.baseUrl = baseUrl.TrimEnd('/');
        }

        /// <summary>Asks the server if it is alive. The reply's status is "ok".</summary>
        public Task<HealthReply> Ping()
        {
            return Send<HealthReply>("GET", "/healthz", null);
        }

        // TODO: your endpoints. For example:
        //
        //   public Task<RoomInfo> CreateRoom(CreateRoomRequest request)
        //   {
        //       return Send<RoomInfo>("POST", "/rooms", request);
        //   }

        /// <summary>
        /// Sends one request and returns the reply, read from JSON into T.
        /// body is sent as JSON; pass null for requests without one.
        /// </summary>
        async Task<T> Send<T>(string method, string path, object body)
        {
            using (var request = new UnityWebRequest(baseUrl + path, method))
            {
                request.downloadHandler = new DownloadHandlerBuffer();
                request.timeout = 8; // seconds; never leave a menu hanging on a dead server

                if (body != null)
                {
                    byte[] json = Encoding.UTF8.GetBytes(JsonUtility.ToJson(body));
                    request.uploadHandler = new UploadHandlerRaw(json);
                    request.SetRequestHeader("Content-Type", "application/json");
                }

                // Wait for the request without blocking the game. Called from the main
                // thread, this resumes on the main thread, so it is safe to touch UI after it.
                var finished = new TaskCompletionSource<bool>();
                request.SendWebRequest().completed += _ => finished.SetResult(true);
                await finished.Task;

                string text = request.downloadHandler.text;

                if (request.result == UnityWebRequest.Result.Success)
                {
                    return JsonUtility.FromJson<T>(text);
                }

                // The server answered with an error: read its code and message.
                if (request.result == UnityWebRequest.Result.ProtocolError)
                {
                    var error = new BackendError();
                    try { JsonUtility.FromJsonOverwrite(text, error); } catch (ArgumentException) { }
                    throw new BackendException(request.responseCode,
                        string.IsNullOrEmpty(error.error) ? "http_" + request.responseCode : error.error,
                        string.IsNullOrEmpty(error.message) ? request.error : error.message);
                }

                // No answer at all: server down, no internet, or timed out.
                throw new BackendException(0, "unreachable", request.error);
            }
        }
    }
}
