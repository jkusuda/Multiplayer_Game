// Week 1 connectivity check. Put this on any GameObject, start the backend, press Play,
// and look at the Console. Delete it once the real menu talks to the backend.
using Game.Backend;
using UnityEngine;

public class BackendPingExample : MonoBehaviour
{
    [SerializeField] string backendUrl = "http://localhost:8080";

    async void Start()
    {
        var backend = new BackendClient(backendUrl);
        try
        {
            HealthReply reply = await backend.Ping();
            Debug.Log("Backend says: " + reply.status);
        }
        catch (BackendException e)
        {
            Debug.LogWarning("Backend call failed: " + e.Code + " (" + e.Message + ")");
        }
    }
}
