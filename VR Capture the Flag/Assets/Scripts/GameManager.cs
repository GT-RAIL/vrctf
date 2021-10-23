using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.Networking;
using System;

public class GameManager : MonoBehaviour
{
    private int OculusID;
    private string TeamName;
    private GSRequest statusUpdate;

    // Start is called before the first frame update
    void Start()
    {
        
        // Start by figuring out which team I'm on: 
        // Generate an OculusID and send to http://server/register
        // The reply will include the OculusIDs and Team (either 0 or 1)
        
        var now = System.DateTime.UtcNow.ToBinary().ToString();
        // Debug.Log(now);
        // Debug.Log(now.Substring(now.Length-7));
        int OculusID = int.Parse(now.Substring(now.Length-7));
        Debug.Log("OculusID = " + OculusID.ToString());

        StartCoroutine(GetTeamNumber(OculusID));
        // Now we know what team we're on (0=Blue, 1=Red). We won't change this from here on out

        // Instantiate ghost players, which are the user-movable ones
    }

    // Update is called once per frame
    void Update()
    {
        // Grab location data for both team's players
        StartCoroutine(GetRequest("http://143.215.60.21:1001/get")); 

        
        

        // Check for user input: new waypoints? (Don't do that during Update, actually. 
        // Have a trigger handler that selects the VR cozmo and do a drag-and-drop to a new location)
        // Option 1: While cozmos are not being held, grab their position information to send to the server as waypoints
        // (easier than Option 2, although perhaps less efficient)
        // Option 2: Update the server only after a cozmo has been picked up, dropped, and stopped moving.

        // Send the location of the new waypoints

        // Update the board in VR (ghost players show actual locations)
    }


    IEnumerator GetRequest(string uri)
    {
        using (UnityWebRequest webRequest = UnityWebRequest.Get(uri))
        {
            // Request and wait for the desired page.
            yield return webRequest.SendWebRequest();

            string[] pages = uri.Split('/');
            int page = pages.Length - 1;

            switch (webRequest.result)
            {
                case UnityWebRequest.Result.ConnectionError:
                case UnityWebRequest.Result.DataProcessingError:
                    Debug.LogError(pages[page] + ": Error: " + webRequest.error);
                    break;
                case UnityWebRequest.Result.ProtocolError:
                    Debug.LogError(pages[page] + ": HTTP Error: " + webRequest.error);
                    break;
                case UnityWebRequest.Result.Success:
                    Debug.Log(pages[page] + ":\nReceived: " + webRequest.downloadHandler.text);
                    GSRequest statusUpdate = JsonUtility.FromJson<GSRequest>(webRequest.downloadHandler.text);
                    break;
            }
        }
    }

    IEnumerator GetTeamNumber(int HeadsetID)
    {
        WWWForm form = new WWWForm();

        form.AddField("OculusId", HeadsetID);

        UnityWebRequest www = UnityWebRequest.Post("http://143.215.60.21:1001/register", form);
        yield return www.SendWebRequest();

        if (www.result != UnityWebRequest.Result.Success)
        {
            Debug.Log(www.error);
        }
        else
        {
            Debug.Log("Form upload complete!\n" + www.downloadHandler.text);
            GSRequest statusUpdate = JsonUtility.FromJson<GSRequest>(www.downloadHandler.text);
            if (statusUpdate.Status == 200){
                Debug.Log("Team is logged in!");
                // Parse return value to determine team: either 0 or 1
                int TeamID = statusUpdate.Team;
                if (TeamID == 0){
                    TeamName = "BlueTeam";
                    Debug.Log("TeamID is " + TeamID + ". You are the " + TeamName);
                    
                }
                if (TeamID == 1){
                    TeamName = "RedTeam";
                    Debug.Log("TeamID is " + TeamID + ". You are the " + TeamName);
                }
            }
            else {
                Debug.Log("Error registering: code " + statusUpdate.Status.ToString());
            }
        }
    }

IEnumerator SendWaypoints(string TeamName, int robot_num, float[] target_pose)
    {
        WWWForm form = new WWWForm();
        // Team: (0,1)
        // Robot: e.g. cozmo_1
        // Field: Location OR Waypoint
        // Value: [x,y] where (x,y) e [0,1]
        // OculusID: int

        string RobotName = "cozmo_" + robot_num.ToString();


        form.AddField("Team", TeamName);
        form.AddField("Robot", RobotName);
        form.AddField("Field", "Waypoint");
        form.AddField("Value", target_pose.ToString());
        form.AddField("OculusID", OculusID.ToString());

        // GSRequest newStatus = new GSRequest();
        // newStatus.Team = TeamID;
        // newStatus.Robot = RobotName;
        // newStatus.Field = "Waypoint";
        // newStatus.Value = target_pose;
        // newStatus.OculusId = OculusID;

        // string json = JsonUtility.ToJson(newStatus);


        UnityWebRequest www = UnityWebRequest.Post("http://143.215.60.21:1001/put", form);
        yield return www.SendWebRequest();

        if (www.result != UnityWebRequest.Result.Success)
        {
            Debug.Log(www.error);
        }
        else
        {
            Debug.Log("Form upload complete!");
        }
    }
        


}

[System.Serializable]
public class GSRequest
{
    public int Team;
    public string Robot;
    public string Field;
    public float[] Value;
    public int OculusId;
    public int Status;

    public static GSRequest CreateFromJSON(string jsonString)
    {
        return JsonUtility.FromJson<GSRequest>(jsonString);
    }

    public string SaveToString()
    {
        return JsonUtility.ToJson(this);
    }

}