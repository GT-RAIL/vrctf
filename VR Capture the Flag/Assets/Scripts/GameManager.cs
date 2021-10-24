using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.Networking;
using System;
using TMPro;

public class GameManager : MonoBehaviour
{
    public int OculusID;
    public string TeamName;
    private JSONObject statusUpdate;

    private List<GameObject> MyTeam = new List<GameObject>();
    private List<GameObject> MyGhostTeam = new List<GameObject>();
    private List<GameObject> OpposingTeam = new List<GameObject>();

    public GameObject waypointPrefab;
    public GameObject redFlagPrefab;
    public GameObject blueFlagPrefab;

    public GameObject redFlag;
    public GameObject blueFlag;

    public TextMeshPro redScoreText;
    public TextMeshPro blueScoreText;


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

        // instantiate the flags at the base
        redFlag = Instantiate(redFlagPrefab, GameObject.Find("RedGoal").transform.position, GameObject.Find("RedGoal").transform.rotation);
        blueFlag = Instantiate(blueFlagPrefab, GameObject.Find("BlueGoal").transform.position, GameObject.Find("RedGoal").transform.rotation);

        redScoreText = GameObject.Find("Red Score Text").GetComponent<TextMeshPro>();
        blueScoreText = GameObject.Find("Blue Score Text").GetComponent<TextMeshPro>();

        // Collect players (ghost players are the user-movable ones)
        for (int i=1;i<=4;i++){
            
            GameObject playerObj = GameObject.Find("anki_cozmo ("+i.ToString()+")");
            MyTeam.Add(playerObj);
            // Debug.Log("My team member added: "+playerObj.name);

            playerObj = GameObject.Find("anki_cozmo_ghost ("+i.ToString()+")");
            MyGhostTeam.Add(playerObj);
            // Debug.Log("My ghost team member added: "+playerObj.name);

            playerObj = GameObject.Find("anki_cozmo ("+(i+4).ToString()+")");
            OpposingTeam.Add(playerObj);
            // Debug.Log("Opposing team member added: "+playerObj.name);
        }
        
        StartCoroutine(LocationCycle());
        
    }

    // Update is called once per frame
    void Update()
    {
        // Use OVRInput and IsGrabbable Script to notice when a hand has grabbed a robot. Flag that robot and the
        // hand that's moving it. Upon release, (wait half a second and) update the waypoint location (send via http:.../put)
        
        // OVRGrabber leftHand = ;

    }

    IEnumerator LocationCycle(){
        while(true){
            StartCoroutine(GetRequest("http://1f82-2610-148-1f02-3000-b082-3101-7d86-202.ngrok.io/get")); 
            yield return new WaitForSeconds(0.1f);

            float offset_x =  -0.5f;
            float offset_y = 0.25f;

            // flags at bases
            if (statusUpdate["RedFlagAtBase"].b) {
                redFlag.transform.position = GameObject.Find("RedGoal").transform.position;
            }
            if (statusUpdate["BlueFlagAtBase"].b) {
                blueFlag.transform.position = GameObject.Find("BlueGoal").transform.position;
            }

            // base locations
            float blue_location_x = (float) statusUpdate["BlueFlagBaseLocation"][0].n;
            float blue_location_y = (float) statusUpdate["BlueFlagBaseLocation"][1].n;
            GameObject.Find("BlueGoal").transform.position = new Vector3(blue_location_x + offset_x, .75f, blue_location_y + offset_y);

            float red_location_x = (float) statusUpdate["RedFlagBaseLocation"][0].n;
            float red_location_y = (float) statusUpdate["RedFlagBaseLocation"][1].n;
            GameObject.Find("RedGoal").transform.position = new Vector3(red_location_x + offset_x, .75f, red_location_y + offset_y);

            redScoreText.SetText("Red Score: " + ((int) statusUpdate["RedTeamScore"].n));
            blueScoreText.SetText("Blue Score: " + ((int) statusUpdate["BlueTeamScore"].n));

            GameObject go;
            if (TeamName == "BlueTeam"){
                for (int i = 1; i<=4; i++){
                    // ignore if the robot does not exist in the data sent
                    if (!statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]) {
                        //Debug.Log("No data for cozmo_" + i.ToString());
                        continue;
                    }

                    // get the game object for the robot
                    go = GameObject.Find("anki_cozmo ( " + i.ToString() + ")");

                    // update the robot's location
                    float location_x = (float) statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["Location"][0].n;
                    float location_y = (float) statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["Location"][1].n;
                    go.transform.position = new Vector3(location_x + offset_x, 0.75f, location_y + offset_y);

                    // update HasRedFlag
                    if (statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["HasRedFlag"].b) {
                        redFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }
                    
                    // update HasBlueFlag                
                    if (statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["HasBlueFlag"].b) {
                        blueFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }           
                }
        
                for (int i = 5; i<=8; i++) {
                    
                    // update location
                    if (!statusUpdate["RedTeam"]["cozmo_" + i.ToString()]) {
                        //Debug.Log("No data for cozmo_" + i.ToString());
                        continue;
                    }

                    go = GameObject.Find("anki_cozmo ( " + i.ToString() + ")");

                    // update the robot's location
                    float location_x = (float) statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["Location"][0].n;
                    float location_y = (float) statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["Location"][1].n;
                    go.transform.position = new Vector3(location_x + offset_x, 0.75f, location_y + offset_y);

                    // update HasRedFlag
                    if (statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["HasRedFlag"].b) {
                        redFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }
                    
                    // update HasBlueFlag                
                    if (statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["HasBlueFlag"].b) {
                        blueFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }  

                }
            }
            if (TeamName == "RedTeam"){
                for (int i = 1; i<=4; i++){
                    // ignore if the robot does not exist in the data sent
                    if (!statusUpdate["RedTeam"]["cozmo_" + i.ToString()]) {
                        //Debug.Log("No data for cozmo_" + i.ToString());
                        continue;
                    }

                    // get the game object for the robot
                    go = GameObject.Find("anki_cozmo ( " + i.ToString() + ")");

                    // update the robot's location
                    float location_x = (float) statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["Location"][0].n;
                    float location_y = (float) statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["Location"][1].n;
                    go.transform.position = new Vector3(location_x + offset_x, 0.75f, location_y + offset_y);

                    // update HasRedFlag
                    if (statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["HasRedFlag"].b) {
                        redFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }
                    
                    // update HasBlueFlag                
                    if (statusUpdate["RedTeam"]["cozmo_" + i.ToString()]["HasBlueFlag"].b) {
                        blueFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }           
                }
        
                for (int i = 5; i<=8; i++) {
                    
                    // update location
                    if (!statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]) {
                        //Debug.Log("No data for cozmo_" + i.ToString());
                        continue;
                    }

                    go = GameObject.Find("anki_cozmo ( " + i.ToString() + ")");

                    // update the robot's location
                    float location_x = (float) statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["Location"][0].n;
                    float location_y = (float) statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["Location"][1].n;
                    go.transform.position = new Vector3(location_x + offset_x, 0.75f, location_y + offset_y);

                    // update HasRedFlag
                    if (statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["HasRedFlag"].b) {
                        redFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }
                    
                    // update HasBlueFlag                
                    if (statusUpdate["BlueTeam"]["cozmo_" + i.ToString()]["HasBlueFlag"].b) {
                        blueFlag.transform.position = new Vector3(location_x + offset_x, 0.85f, location_y + offset_y);
                    }  

                }
            }
        }
    }

    // from https://forum.unity.com/threads/turn-string-into-list-of-ints.340341/
    public List<float> GetFloatsFromString(string str){
        str = str.Substring(1, str.Length);
        List<float> floats = new List<float>();
     
        string[] splitString = str.Split(new string[] { "," }, StringSplitOptions.RemoveEmptyEntries);
        foreach (string item in splitString)
        {
            try
            {
                floats.Add(float.Parse(item));
            }
            catch (System.Exception e)
            {
                Debug.LogError("Value in string was not an int.");
                Debug.LogException(e);
            }
        }
        return floats;
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
                    statusUpdate = new JSONObject(webRequest.downloadHandler.text); //JsonUtility.FromJson<GSRequest>(webRequest.downloadHandler.text);
                    break;
            }
        }
    }

    IEnumerator GetTeamNumber(int HeadsetID)
    {
        WWWForm form = new WWWForm();

        form.AddField("OculusId", HeadsetID);

        UnityWebRequest www = UnityWebRequest.Post("http://1f82-2610-148-1f02-3000-b082-3101-7d86-202.ngrok.io/register", form);
        yield return www.SendWebRequest();

        if (www.result != UnityWebRequest.Result.Success)
        {
            Debug.Log(www.error);
        }
        else
        {
            Debug.Log("Form upload complete!\n" + www.downloadHandler.text);
            GSRequest TeamIDUpdate = JsonUtility.FromJson<GSRequest>(www.downloadHandler.text);
            if (TeamIDUpdate.Status == 200){
                Debug.Log("Team is logged in!");
                // Parse return value to determine team: either 0 or 1
                int TeamID = TeamIDUpdate.Team;
                if (TeamID == 0){
                    TeamName = "BlueTeam";
                    Debug.Log("TeamID is " + TeamID + ". You are the " + TeamName);
                    GameObject.Find("Team Text").GetComponent<TextMeshPro>().SetText("Team: " + TeamName);
                    
                }
                if (TeamID == 1){
                    TeamName = "RedTeam";
                    Debug.Log("TeamID is " + TeamID + ". You are the " + TeamName);
                    GameObject.Find("Team Text").GetComponent<TextMeshPro>().SetText("Team: " + TeamName);
                }
            }
            else {
                Debug.Log("Error registering: code " + TeamIDUpdate.Status.ToString());
            }
        }
    }
 

    [System.Serializable]
    public class CozmoStats
    {
        public List<float> Location;
        public List<float> Waypoint;
        public bool HasRedFlag;
        public bool HasBlueFlag;
        public bool CanMove;
        public float AuraCount;
    }

    [System.Serializable]
    public class GSRequest
    {
        public int Team;
        public string Robot;
        public string Field;
        public List<float> Value;
        public int OculusId;
        public int Status;

        public bool BlueFlagAtBase;
        public List<float> BlueFlagBaseLocation;
        public bool RedFlagAtBase;
        public List<float> RedFlagBaseLocation;
        public Dictionary<string, CozmoStats> BlueTeam; 
        public Dictionary<string, CozmoStats> RedTeam; 

        public int RedTeamScore;
        public int BlueTeamScore;

    }
}