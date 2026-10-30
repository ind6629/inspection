using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

[Serializable] public class IssuePosition { public float x,y,z; public Vector3 Vector => new Vector3(x,y,z); public IssuePosition(Vector3 v){x=v.x;y=v.y;z=v.z;} }
[Serializable] public class CreateIssue { public string title,description,priority; public IssuePosition position; }
[Serializable] public class Issue { public string id,title,description,priority,status,createdAt,updatedAt; public IssuePosition position; }
[Serializable] public class IssueList { public Issue[] items; }
[Serializable] public class ApiError { public ErrorMessage error; }
[Serializable] public class ErrorMessage { public string message; }
public class InspectionMarker : MonoBehaviour { public Issue issue; }

// IMGUI owns all interactive UI; screen rectangles explicitly block world clicks.
public class InspectionApp : MonoBehaviour
{
    public string backendUrl = "http://localhost:8080";
    Camera view;
    CharacterController player;
    float pitch, yaw;
    GameObject pending;
    Vector3 hitPosition;
    bool formOpen, submitting, loading;
    string title="", description="", message="Ready. Click equipment to report an issue.";
    int priority=1;
    readonly string[] priorities={"low","medium","high"};
    readonly Dictionary<string,InspectionMarker> markers=new Dictionary<string,InspectionMarker>();
    Issue selected;
    Material surfaceMaterial;
    Rect Header => new Rect(16,16,440,160);
    Rect Modal => new Rect((Screen.width-480)/2f, Mathf.Max(20,(Screen.height-480)/2f),480,480);

    [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]
    static void Bootstrap(){if(FindFirstObjectByType<InspectionApp>()==null)new GameObject("InspectionApp").AddComponent<InspectionApp>();}

    void Start()
    {
        surfaceMaterial=Resources.Load<Material>("SiteSurface");
        if(surfaceMaterial==null||!surfaceMaterial.shader.isSupported)throw new Exception("SiteSurface material missing or unsupported");
        // Fixed world layout: coordinates remain meaningful between desktop sessions.
        foreach(var camera in FindObjectsByType<Camera>(FindObjectsSortMode.None)) camera.gameObject.SetActive(false);
        RenderSettings.ambientLight=new Color(.64f,.69f,.75f);
        var lightObject=new GameObject("Sun");var light=lightObject.AddComponent<Light>();light.type=LightType.Directional;light.intensity=1.3f;lightObject.transform.rotation=Quaternion.Euler(50,-30,0);
        Box("Floor",new Vector3(0,-.15f,0),new Vector3(18,.3f,18),new Color(.3f,.36f,.4f));
        Box("Back wall",new Vector3(0,2,8),new Vector3(18,4,.3f),new Color(.64f,.7f,.73f));
        Box("Left wall",new Vector3(-9,2,0),new Vector3(.3f,4,16),new Color(.55f,.62f,.67f));
        for(int i=0;i<3;i++){
            float x=-4+i*4;
            Box("Cabinet "+(i+1),new Vector3(x,1.5f,3),new Vector3(1.8f,3,1.4f),new Color(.12f,.23f,.3f));
            for(int j=0;j<6;j++)Box("Equipment panel",new Vector3(x,.45f+j*.42f,2.28f),new Vector3(1.5f,.28f,.05f),new Color(.25f,.42f,.49f));
            var label=new GameObject("Cabinet label");label.transform.position=new Vector3(x,3.3f,2.2f);var text=label.AddComponent<TextMesh>();text.text="CABINET 0"+(i+1);text.characterSize=.14f;text.fontSize=40;text.anchor=TextAnchor.MiddleCenter;text.color=Color.white;
        }
        Box("Pipe",new Vector3(0,3.7f,6),new Vector3(14,.2f,.2f),new Color(.9f,.58f,.16f));
        var body=new GameObject("Player");body.transform.position=new Vector3(0,0,-5);player=body.AddComponent<CharacterController>();player.height=1.8f;player.center=new Vector3(0,.9f,0);player.radius=.3f;
        var cameraObject=new GameObject("Inspection Camera");cameraObject.transform.SetParent(body.transform);cameraObject.transform.localPosition=new Vector3(0,1.65f,0);view=cameraObject.AddComponent<Camera>();view.nearClipPlane=.05f;view.backgroundColor=new Color(.12f,.19f,.25f);view.clearFlags=CameraClearFlags.SolidColor;
        StartCoroutine(Refresh());
        if(Array.IndexOf(Environment.GetCommandLineArgs(),"-inspection-smoke")>=0)StartCoroutine(Smoke());
        var args=Environment.GetCommandLineArgs();int preview=Array.IndexOf(args,"-inspection-preview");
        if(preview>=0&&preview+1<args.Length)StartCoroutine(CapturePreview(args[preview+1]));
    }
    void SetSurface(GameObject obj,Color color){var material=new Material(surfaceMaterial);material.color=color;obj.GetComponent<Renderer>().sharedMaterial=material;}
    GameObject Box(string name,Vector3 position,Vector3 scale,Color color){var obj=GameObject.CreatePrimitive(PrimitiveType.Cube);obj.name=name;obj.transform.position=position;obj.transform.localScale=scale;obj.layer=8;SetSurface(obj,color);return obj;}
    void Update()
    {
        if(view==null)return;
        bool modal=formOpen||selected!=null;
        Vector2 pointer=new Vector2(Input.mousePosition.x,Screen.height-Input.mousePosition.y);
        bool overUI=Header.Contains(pointer)||(modal&&Modal.Contains(pointer));
        if(!modal&&!submitting){
            if(Input.GetMouseButton(1)&&!overUI){yaw+=Input.GetAxis("Mouse X")*2.2f;pitch=Mathf.Clamp(pitch-Input.GetAxis("Mouse Y")*2.2f,-75,75);player.transform.rotation=Quaternion.Euler(0,yaw,0);view.transform.localRotation=Quaternion.Euler(pitch,0,0);}
            Vector3 move=player.transform.right*Input.GetAxisRaw("Horizontal")+player.transform.forward*Input.GetAxisRaw("Vertical");move=Vector3.ClampMagnitude(move,1)*4;move.y=-9.8f;player.Move(move*Time.deltaTime);
        }
        if(modal||submitting||loading||overUI||!Input.GetMouseButtonDown(0))return;
        Ray ray=view.ScreenPointToRay(Input.mousePosition);
        if(!Physics.Raycast(ray,out RaycastHit hit,100f, (1<<8)|(1<<9),QueryTriggerInteraction.Ignore))return;
        var marker=hit.collider.GetComponent<InspectionMarker>();
        if(marker!=null){selected=marker.issue;return;}
        if(pending!=null)return;
        hitPosition=hit.point;
        pending=MakeMarker(hit.point,Color.yellow);pending.name="Pending issue";
        formOpen=true;title="";description="";priority=1;
    }
    GameObject MakeMarker(Vector3 position,Color color){var obj=GameObject.CreatePrimitive(PrimitiveType.Sphere);obj.transform.position=position;obj.transform.localScale=Vector3.one*.22f;obj.layer=9;SetSurface(obj,color);return obj;}
    Color MarkerColor(Issue issue){return issue.status=="resolved"?Color.green:issue.priority=="high"?new Color(1,.25f,.2f):issue.priority=="medium"?new Color(1,.65f,.1f):Color.cyan;}
    void PutMarker(Issue issue){if(markers.TryGetValue(issue.id,out var marker)){marker.issue=issue;marker.GetComponent<Renderer>().material.color=MarkerColor(issue);return;}var obj=MakeMarker(issue.position.Vector,MarkerColor(issue));obj.name="Issue "+issue.id;marker=obj.AddComponent<InspectionMarker>();marker.issue=issue;markers.Add(issue.id,marker);}
    void OnGUI()
    {
        GUI.skin.label.wordWrap=true;
        GUILayout.BeginArea(Header,GUI.skin.box);
        GUILayout.Label("VIRTUAL SITE INSPECTION");GUILayout.Label("WASD: move | Hold RMB: look | LMB: mark / inspect");
        GUILayout.BeginHorizontal();GUILayout.Label("Backend",GUILayout.Width(65));GUI.enabled=!submitting&&!loading;backendUrl=GUILayout.TextField(backendUrl);GUILayout.EndHorizontal();
        GUI.enabled=!loading&&!submitting&&!formOpen&&selected==null;if(GUILayout.Button(loading?"Loading...":"Reload saved markers"))StartCoroutine(Refresh());GUI.enabled=true;
        GUILayout.Label(message);GUILayout.EndArea();
        if(formOpen){GUILayout.BeginArea(Modal,GUI.skin.window);GUILayout.Label("REPORT AN ISSUE");GUILayout.Label("Hit position: "+hitPosition.ToString("F3"));
            GUI.enabled=!submitting;
            GUILayout.Label("Title (required)");title=GUILayout.TextField(title,160);GUILayout.Label("Description (optional)");description=GUILayout.TextArea(description,2000,GUILayout.Height(100));GUILayout.Label("Priority");priority=GUILayout.SelectionGrid(priority,priorities,3);
            GUILayout.Space(15);if(GUILayout.Button(submitting?"Submitting...":"Submit issue",GUILayout.Height(36))){if(string.IsNullOrWhiteSpace(title))message="Title is required.";else StartCoroutine(Submit());}
            if(GUILayout.Button("Cancel")){Destroy(pending);pending=null;formOpen=false;message="Cancelled. Unsaved marker removed.";}GUI.enabled=true;GUILayout.Label(message);GUILayout.EndArea();
        }else if(selected!=null){GUILayout.BeginArea(Modal,GUI.skin.window);GUILayout.Label("SAVED ISSUE");GUILayout.Label(selected.title);GUILayout.Label("Priority: "+selected.priority+" | Status: "+selected.status);GUILayout.Label(selected.description);GUILayout.Label("Position: "+selected.position.Vector.ToString("F3"));GUILayout.Label("ID: "+selected.id);GUILayout.Space(20);if(GUILayout.Button("Close"))selected=null;GUILayout.EndArea();}
    }
    IEnumerator Refresh()
    {
        loading=true;message="Loading saved markers...";
        using(var request=UnityWebRequest.Get(backendUrl.TrimEnd('/')+"/api/issues")){
            request.timeout=10;yield return request.SendWebRequest();
            if(request.result!=UnityWebRequest.Result.Success)message="Cannot load backend. Start Go, then reload. "+request.error;
            else {try{var list=JsonUtility.FromJson<IssueList>("{\"items\":"+request.downloadHandler.text+"}");foreach(var issue in list.items)PutMarker(issue);message="Loaded "+list.items.Length+" saved issue(s).";}catch(Exception){message="Invalid backend response.";}}
        }loading=false;
    }
    IEnumerator Submit()
    {
        submitting=true;message="Submitting...";
        var input=new CreateIssue{title=title.Trim(),description=description.Trim(),priority=priorities[priority],position=new IssuePosition(hitPosition)};
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/issues","POST")){
            request.uploadHandler=new UploadHandlerRaw(Encoding.UTF8.GetBytes(JsonUtility.ToJson(input)));request.downloadHandler=new DownloadHandlerBuffer();request.SetRequestHeader("Content-Type","application/json");request.timeout=10;
            yield return request.SendWebRequest();
            if(request.result==UnityWebRequest.Result.Success){try{var issue=JsonUtility.FromJson<Issue>(request.downloadHandler.text);if(string.IsNullOrEmpty(issue.id)||issue.position==null)throw new Exception();PutMarker(issue);Destroy(pending);pending=null;formOpen=false;message="Issue saved. Refresh React to view it.";}catch(Exception){message="Invalid response. Check admin before retrying.";}}
            else{message="Submit failed: "+request.error+". Check admin before retrying.";try{var err=JsonUtility.FromJson<ApiError>(request.downloadHandler.text);if(err?.error!=null)message=err.error.message;}catch(Exception){}}
        }submitting=false;
    }
    IEnumerator CapturePreview(string path){yield return new WaitForSeconds(2);yield return new WaitForEndOfFrame();ScreenCapture.CaptureScreenshot(path);Debug.Log("INSPECTION_PREVIEW_REQUESTED");}
    // Opt-in integration check for CI/local validation; never runs in normal play.
    IEnumerator Smoke()
    {
        float deadline=Time.realtimeSinceStartup+20;
        while(loading&&Time.realtimeSinceStartup<deadline)yield return null;
        yield return new WaitForFixedUpdate();
        if(!Physics.Raycast(view.ViewportPointToRay(new Vector3(.5f,.5f)),out var hit,100f,1<<8)){
            Debug.LogError("SMOKE_FAIL: scene ray missed");Application.Quit(1);yield break;
        }
        hitPosition=hit.point;pending=MakeMarker(hitPosition,Color.yellow);formOpen=true;
        title="Unity runtime integration check";description="Created by opt-in smoke test";priority=2;
        yield return Submit();
        if(formOpen||pending!=null){Debug.LogError("SMOKE_FAIL: "+message);Application.Quit(1);yield break;}
        yield return Refresh();
        if(markers.Count==0){Debug.LogError("SMOKE_FAIL: saved marker missing");Application.Quit(1);yield break;}
        Debug.Log("INSPECTION_SMOKE_PASS: physics ray -> UnityWebRequest -> Go save -> reload markers");
        Application.Quit(0);
    }
}
