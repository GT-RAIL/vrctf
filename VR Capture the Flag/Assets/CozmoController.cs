using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using TMPro;
using UnityEngine.Networking;
using System;

public class CozmoController : MonoBehaviour
{

    public GameObject linkedGhost;
    public string name;
    public string oldTag;
    public TextMeshPro debugMessage;
    public GameManager gameManager;

    public Vector2 waypoint;
    public GameObject waypointMarker;

    // Start is called before the first frame update
    void Start()
    {
        gameManager = GameObject.Find("PlayerManager").GetComponent<GameManager>();
        name = "cozmo_1";
        oldTag = transform.tag;
        debugMessage = GameObject.Find("Team Text").GetComponent<TextMeshPro>();
        waypointMarker = Instantiate(gameManager.waypointPrefab, transform.position, transform.rotation, transform);
    }

    // Update is called once per frame
    void Update()
    {
        if (linkedGhost.transform.tag == "picked" && linkedGhost.transform.tag != oldTag) {
            debugMessage.SetText("PICKED!");
            oldTag = linkedGhost.transform.tag;
        }
        if (linkedGhost.transform.tag != "picked" && linkedGhost.transform.tag != oldTag) {
            debugMessage.SetText("released");
            oldTag = linkedGhost.transform.tag;
            StartCoroutine(SendWaypoint());  // send the waypoint to the server
        }
    }

    IEnumerator SendWaypoint()
    {
        Debug.Log("Now sending waypoints.");
        WWWForm form = new WWWForm();
        // Team: (0,1)
        // Robot: e.g. cozmo_1
        // Field: Location OR Waypoint
        // Value: [x,y] where (x,y) e [0,1]
        // OculusID: int

        float waypoint_x = linkedGhost.transform.position.x + 0.5f;
        float waypoint_y = linkedGhost.transform.position.z - 0.25f;
        waypoint = new Vector2(waypoint_x, waypoint_y);
        waypointMarker.transform.position = new Vector3(linkedGhost.transform.position.x, 0.75f, linkedGhost.transform.position.z);

        form.AddField("Team", gameManager.TeamName);
        form.AddField("Robot", name);
        form.AddField("Field", "Waypoint");
        form.AddField("Value", "[" + waypoint_x + "," + waypoint_y + "]");
        form.AddField("OculusId", gameManager.OculusID);

        UnityWebRequest www = UnityWebRequest.Post("http://1f82-2610-148-1f02-3000-b082-3101-7d86-202.ngrok.io/put", form);
        yield return www.SendWebRequest();

        if (www.result != UnityWebRequest.Result.Success) { Debug.Log(www.error); }
        else { Debug.Log("added waypoint!"); }

        // send the ghost back to the robot
        linkedGhost.transform.position = transform.position;
        linkedGhost.transform.rotation = transform.rotation;
    }

}
