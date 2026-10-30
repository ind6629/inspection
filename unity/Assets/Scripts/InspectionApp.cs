using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

[Serializable] public class IssuePosition { public float x,y,z; public Vector3 Vector => new Vector3(x,y,z); public IssuePosition(Vector3 v){x=v.x;y=v.y;z=v.z;} }
[Serializable] public class CreateIssue { public string title,description,priority,deviceId,area; public IssuePosition position; }
[Serializable] public class PasswordChangeInput { public string currentPassword,newPassword; }
[Serializable] public class Reporter { public string id,username,displayName; }
[Serializable] public class LoginInput { public string username,password; }
[Serializable] public class RegisterInput { public string username,displayName,password; }
[Serializable] public class InspectorLogin { public string token,expiresAt; public Reporter user; }
[Serializable] public class Issue { public string id,title,description,priority,status,createdAt,updatedAt,deviceId,area; public IssuePosition position; public Reporter reporter; }
[Serializable] public class IssueList { public Issue[] items; }
[Serializable] public class ApiError { public ErrorMessage error; }
[Serializable] public class ErrorMessage { public string message; }
public class InspectionMarker : MonoBehaviour { public Issue issue; }
public class InspectionSurface : MonoBehaviour { public string deviceId,area; }

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
    string title="", description="", message="准备就绪，点击设备或地面即可上报问题。";
    int priority=1;
    readonly string[] priorities={"low","medium","high"};
    readonly string[] priorityLabels={"低优先级","中优先级","高优先级"};
    readonly Dictionary<string,InspectionMarker> markers=new Dictionary<string,InspectionMarker>();
    Issue selected;
    Material surfaceMaterial;
    Font chineseFont;
    string loginUsername="inspector01",loginPassword="",authToken="";
    Reporter currentUser;
    bool loggingIn;
    bool registerMode;
    string registerName="",confirmPassword="";
    string pendingDeviceId="",pendingArea="";
    bool passwordOpen;
    string oldPassword="",newPassword="",newPasswordConfirmation="";
    Rect LoginRect { get {float height=registerMode?660:570;return new Rect((Screen.width-520)/2f,Mathf.Max(16,(Screen.height-height)/2f),520,height);} }
    Rect Header => new Rect(16,16,520,250);
    Rect Modal => new Rect((Screen.width-520)/2f, Mathf.Max(20,(Screen.height-540)/2f),520,540);

    [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]
    static void Bootstrap(){if(FindFirstObjectByType<InspectionApp>()==null)new GameObject("InspectionApp").AddComponent<InspectionApp>();}

    void Start()
    {
        if(Array.IndexOf(Environment.GetCommandLineArgs(),"-inspection-smoke")>=0)backendUrl=Environment.GetEnvironmentVariable("INSPECTION_SMOKE_BACKEND")??backendUrl;
        // Use the installed Windows font without redistributing a proprietary font file.
        chineseFont=Font.CreateDynamicFontFromOSFont(new[]{"Microsoft YaHei","SimHei","Noto Sans CJK SC","Arial Unicode MS"},18);
        surfaceMaterial=Resources.Load<Material>("SiteSurface");
        if(surfaceMaterial==null||(SystemInfo.graphicsDeviceType!=UnityEngine.Rendering.GraphicsDeviceType.Null&&!surfaceMaterial.shader.isSupported))throw new Exception("SiteSurface material missing or unsupported");
        // Fixed world layout: coordinates remain meaningful between desktop sessions.
        foreach(var camera in FindObjectsByType<Camera>(FindObjectsSortMode.None)) camera.gameObject.SetActive(false);
        RenderSettings.ambientLight=new Color(.64f,.69f,.75f);
        var lightObject=new GameObject("Sun");var light=lightObject.AddComponent<Light>();light.type=LightType.Directional;light.intensity=1.3f;lightObject.transform.rotation=Quaternion.Euler(50,-30,0);
        Box("Floor",new Vector3(0,-.15f,0),new Vector3(18,.3f,18),new Color(.3f,.36f,.4f));
        Box("Back wall",new Vector3(0,2,8),new Vector3(18,4,.3f),new Color(.64f,.7f,.73f));
        Box("Left wall",new Vector3(-9,2,0),new Vector3(.3f,4,16),new Color(.55f,.62f,.67f));
        for(int i=0;i<3;i++){
            float x=-4+i*4;
            string device="CAB-"+(i+1).ToString("000"),area=i<2?"机房A区":"机房B区";
            Box("Cabinet "+(i+1),new Vector3(x,1.5f,3),new Vector3(1.8f,3,1.4f),new Color(.12f,.23f,.3f),device,area);
            for(int j=0;j<6;j++)Box("Equipment panel",new Vector3(x,.45f+j*.42f,2.28f),new Vector3(1.5f,.28f,.05f),new Color(.25f,.42f,.49f),device,area);
            var label=new GameObject("Cabinet label");label.transform.position=new Vector3(x,3.5f,2.2f);var text=label.AddComponent<TextMesh>();text.font=chineseFont;label.GetComponent<MeshRenderer>().sharedMaterial=chineseFont.material;text.text=area+"\n"+device;text.characterSize=.07f;text.fontSize=40;text.anchor=TextAnchor.MiddleCenter;text.color=Color.white;
        }
        Box("Pipe",new Vector3(0,3.7f,6),new Vector3(14,.2f,.2f),new Color(.9f,.58f,.16f),"PIPE-001","机房公共区");
        var body=new GameObject("Player");body.transform.position=new Vector3(0,0,-5);player=body.AddComponent<CharacterController>();player.height=1.8f;player.center=new Vector3(0,.9f,0);player.radius=.3f;
        var cameraObject=new GameObject("Inspection Camera");cameraObject.transform.SetParent(body.transform);cameraObject.transform.localPosition=new Vector3(0,1.65f,0);view=cameraObject.AddComponent<Camera>();view.nearClipPlane=.05f;view.backgroundColor=new Color(.12f,.19f,.25f);view.clearFlags=CameraClearFlags.SolidColor;
        view.cullingMask=0;
        message="请使用巡检员账号登录。";
        if(Array.IndexOf(Environment.GetCommandLineArgs(),"-inspection-smoke")>=0)StartCoroutine(Smoke());
        var args=Environment.GetCommandLineArgs();int preview=Array.IndexOf(args,"-inspection-preview");
        if(Array.IndexOf(args,"-inspection-preview-register")>=0){registerMode=true;loginUsername="";message="请填写姓名、账号和密码。注册后将自动登录。";}
        if(preview>=0&&preview+1<args.Length)StartCoroutine(CapturePreview(args[preview+1]));
    }
    void SetSurface(GameObject obj,Color color){var material=new Material(surfaceMaterial);material.color=color;obj.GetComponent<Renderer>().sharedMaterial=material;}
    GameObject Box(string name,Vector3 position,Vector3 scale,Color color,string deviceId="",string area=""){var obj=GameObject.CreatePrimitive(PrimitiveType.Cube);obj.name=name;obj.transform.position=position;obj.transform.localScale=scale;obj.layer=8;SetSurface(obj,color);var surface=obj.AddComponent<InspectionSurface>();surface.deviceId=deviceId;surface.area=area;return obj;}
    void SetHitLocation(RaycastHit hit){hitPosition=hit.point;var surface=hit.collider.GetComponentInParent<InspectionSurface>();pendingDeviceId=surface==null?"":surface.deviceId;pendingArea=surface!=null&&!string.IsNullOrEmpty(surface.area)?surface.area:hit.point.x<2?"机房A区":"机房B区";}
    void Update()
    {
        if(view==null||currentUser==null)return;
        bool modal=formOpen||selected!=null||passwordOpen;
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
        SetHitLocation(hit);
        pending=MakeMarker(hit.point,Color.yellow);pending.name="Pending issue";
        formOpen=true;title="";description="";priority=1;
    }
    GameObject MakeMarker(Vector3 position,Color color){var obj=GameObject.CreatePrimitive(PrimitiveType.Sphere);obj.transform.position=position;obj.transform.localScale=Vector3.one*.22f;obj.layer=9;SetSurface(obj,color);return obj;}
    Color MarkerColor(Issue issue){return issue.status=="resolved"?Color.green:issue.priority=="high"?new Color(1,.25f,.2f):issue.priority=="medium"?new Color(1,.65f,.1f):Color.cyan;}
    void PutMarker(Issue issue){if(markers.TryGetValue(issue.id,out var marker)){marker.issue=issue;marker.GetComponent<Renderer>().material.color=MarkerColor(issue);return;}var obj=MakeMarker(issue.position.Vector,MarkerColor(issue));obj.name="Issue "+issue.id;marker=obj.AddComponent<InspectionMarker>();marker.issue=issue;markers.Add(issue.id,marker);}
    void OnGUI()
    {
        GUI.skin.font=chineseFont;
        GUI.skin.label.fontSize=16;GUI.skin.button.fontSize=16;GUI.skin.textField.fontSize=16;GUI.skin.textArea.fontSize=16;
        GUI.skin.button.fixedHeight=32;GUI.skin.textField.fixedHeight=28;
        GUI.skin.label.wordWrap=true;
        if(currentUser==null){
            GUILayout.BeginArea(LoginRect,GUI.skin.window);
            GUILayout.Label(registerMode?"创建巡检员账号":"虚拟现场巡检 · 巡检员登录");GUILayout.Space(12);
            GUILayout.Label("登录后进入现场，发现的问题将自动记录您的身份。");
            GUI.enabled=!loggingIn;
            GUILayout.Label("后端地址");backendUrl=GUILayout.TextField(backendUrl);
            if(registerMode){GUILayout.Label("姓名 / 显示名（必填）");registerName=GUILayout.TextField(registerName,40);}
            GUILayout.Label("巡检员账号");loginUsername=GUILayout.TextField(loginUsername,100);
            GUILayout.Label("登录密码");loginPassword=GUILayout.PasswordField(loginPassword,'●',100);
            if(registerMode){GUILayout.Label("确认密码");confirmPassword=GUILayout.PasswordField(confirmPassword,'●',100);}
            GUILayout.Space(15);
            if(GUILayout.Button(loggingIn?"正在处理…":registerMode?"注册并进入巡检现场":"登录并进入巡检现场")){
                if(string.IsNullOrWhiteSpace(loginUsername)||string.IsNullOrEmpty(loginPassword))message="请输入巡检员账号和密码。";
                else if(registerMode&&string.IsNullOrWhiteSpace(registerName))message="请填写姓名或显示名。";
                else if(registerMode&&loginPassword!=confirmPassword)message="两次输入的密码不一致。";
                else if(registerMode)StartCoroutine(Register());
                else StartCoroutine(Login());
            }
            if(GUILayout.Button(registerMode?"已有账号？返回登录":"没有账号？注册巡检员")){registerMode=!registerMode;loginUsername="";loginPassword="";confirmPassword="";message=registerMode?"账号3～32位，使用字母、数字、下划线，以字母开头。密码至少6个字符。":"请使用巡检员账号登录。";}
            GUI.enabled=true;GUILayout.Space(12);GUILayout.Label(message);GUILayout.Space(16);
            if(!registerMode)GUILayout.Label("演示账号：inspector01 / inspector02\n密码见后端 data/inspector-credentials.txt 文件。\n管理员账号用于网页管理端。");
            GUILayout.EndArea();return;
        }
        GUILayout.BeginArea(Header,GUI.skin.box);
        GUILayout.Label("虚拟现场巡检");GUILayout.Label("WASD 移动｜按住右键观察｜左键标注或查看问题");
        GUILayout.Label("当前巡检员："+currentUser.displayName+"（"+currentUser.username+"）");
        GUILayout.Label("后端："+backendUrl);
        GUI.enabled=!loading&&!submitting&&!formOpen&&selected==null&&!passwordOpen;if(GUILayout.Button(loading?"正在加载…":"刷新已保存的问题标记"))StartCoroutine(Refresh());GUI.enabled=true;
        GUI.enabled=!loading&&!submitting&&!formOpen&&selected==null&&!passwordOpen;GUILayout.BeginHorizontal();if(GUILayout.Button("修改我的密码")){passwordOpen=true;oldPassword="";newPassword="";newPasswordConfirmation="";message="请输入原密码和新密码。";}if(GUILayout.Button("退出登录 / 切换巡检员"))StartCoroutine(Logout());GUILayout.EndHorizontal();GUI.enabled=true;
        GUILayout.Label(message);GUILayout.EndArea();
        if(passwordOpen||formOpen||selected!=null)DrawModalBackground(Modal);
        if(passwordOpen){GUILayout.BeginArea(Modal,GUI.skin.window);GUILayout.Label("修改我的密码");GUILayout.Label("当前账号："+currentUser.username);GUI.enabled=!submitting;
            GUILayout.Label("原密码");oldPassword=GUILayout.PasswordField(oldPassword,'●',100);GUILayout.Label("新密码（至少6个字符）");newPassword=GUILayout.PasswordField(newPassword,'●',100);GUILayout.Label("确认新密码");newPasswordConfirmation=GUILayout.PasswordField(newPasswordConfirmation,'●',100);GUILayout.Space(16);
            if(GUILayout.Button(submitting?"正在修改…":"确认修改")){if(string.IsNullOrEmpty(oldPassword))message="请输入原密码。";else if(newPassword.Length<6)message="新密码至少6个字符。";else if(newPassword!=newPasswordConfirmation)message="两次新密码不一致。";else StartCoroutine(ChangePassword());}
            if(GUILayout.Button("取消")){passwordOpen=false;oldPassword="";newPassword="";newPasswordConfirmation="";message="已取消修改密码。";}GUI.enabled=true;GUILayout.Label(message);GUILayout.Label("修改成功后需要用新密码重新登录，其他设备的登录也会失效。");GUILayout.EndArea();return;}
        if(formOpen){GUILayout.BeginArea(Modal,GUI.skin.window);GUILayout.Label("上报巡检问题");GUILayout.Label("发现人："+currentUser.displayName+"（后端自动记录）");GUILayout.Label("区域："+pendingArea+"｜设备："+(string.IsNullOrEmpty(pendingDeviceId)?"场景表面（无设备）":pendingDeviceId));GUILayout.Label("标注位置："+hitPosition.ToString("F3"));
            GUI.enabled=!submitting;
            GUILayout.Label("问题标题（必填）");title=GUILayout.TextField(title,160);GUILayout.Label("问题描述（选填）");description=GUILayout.TextArea(description,2000,GUILayout.Height(100));GUILayout.Label("优先级");priority=GUILayout.SelectionGrid(priority,priorityLabels,3);
            GUILayout.Space(15);if(GUILayout.Button(submitting?"正在提交…":"提交问题",GUILayout.Height(36))){if(string.IsNullOrWhiteSpace(title))message="请填写问题标题。";else StartCoroutine(Submit());}
            if(GUILayout.Button("取消填写")){Destroy(pending);pending=null;formOpen=false;message="已取消，未保存的标记已移除。";}GUI.enabled=true;GUILayout.Label(message);GUILayout.EndArea();
        }else if(selected!=null){GUILayout.BeginArea(Modal,GUI.skin.window);GUILayout.Label("问题详情");GUILayout.Label(selected.title);GUILayout.Label("发现人："+(selected.reporter==null?"历史记录（未记录）":selected.reporter.displayName+"（"+selected.reporter.username+"）"));GUILayout.Label("优先级："+PriorityLabel(selected.priority)+"｜状态："+StatusLabel(selected.status));GUILayout.Label(string.IsNullOrWhiteSpace(selected.description)?"暂无描述":selected.description);GUILayout.Label("区域："+(string.IsNullOrEmpty(selected.area)?"未记录":selected.area)+"｜设备："+(string.IsNullOrEmpty(selected.deviceId)?"未记录 / 场景表面":selected.deviceId));GUILayout.Label("位置："+selected.position.Vector.ToString("F3"));GUILayout.Label("问题编号："+selected.id);GUILayout.Space(20);if(GUILayout.Button("关闭详情"))selected=null;GUILayout.EndArea();}
    }
    void ClearSession(string reason){authToken="";currentUser=null;loginPassword="";selected=null;formOpen=false;passwordOpen=false;oldPassword="";newPassword="";newPasswordConfirmation="";if(pending!=null)Destroy(pending);pending=null;foreach(var marker in markers.Values)if(marker!=null)Destroy(marker.gameObject);markers.Clear();if(view!=null)view.cullingMask=0;message=reason;}
    void DrawModalBackground(Rect rect){var original=GUI.color;GUI.color=new Color(0,0,0,.55f);GUI.DrawTexture(new Rect(0,0,Screen.width,Screen.height),Texture2D.whiteTexture);GUI.color=new Color(.10f,.15f,.18f,1);GUI.DrawTexture(rect,Texture2D.whiteTexture);GUI.color=original;}
    void Authorize(UnityWebRequest request){request.SetRequestHeader("Authorization","Bearer "+authToken);}
    IEnumerator ChangePassword(){
        submitting=true;message="正在修改密码…";
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/inspector/password","POST")){
            request.uploadHandler=new UploadHandlerRaw(Encoding.UTF8.GetBytes(JsonUtility.ToJson(new PasswordChangeInput{currentPassword=oldPassword,newPassword=newPassword})));request.downloadHandler=new DownloadHandlerBuffer();request.SetRequestHeader("Content-Type","application/json");Authorize(request);request.timeout=10;yield return request.SendWebRequest();
            if(request.result==UnityWebRequest.Result.Success)ClearSession("密码修改成功，请使用新密码重新登录。");
            else if(request.responseCode==401)ClearSession("登录已失效，请重新登录。");
            else{message="修改失败。若请求超时，请先尝试用新密码登录，确认修改结果。";try{var error=JsonUtility.FromJson<ApiError>(request.downloadHandler.text);if(error?.error!=null)message=error.error.message;}catch(Exception){}}
        }submitting=false;
    }
    IEnumerator Register(){
        loggingIn=true;message="正在创建巡检员账号…";
        bool created=false;
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/inspector/register","POST")){
            var data=new RegisterInput{username=loginUsername.Trim().ToLowerInvariant(),displayName=registerName.Trim(),password=loginPassword};
            request.uploadHandler=new UploadHandlerRaw(Encoding.UTF8.GetBytes(JsonUtility.ToJson(data)));request.downloadHandler=new DownloadHandlerBuffer();request.SetRequestHeader("Content-Type","application/json");request.timeout=10;
            yield return request.SendWebRequest();
            if(request.result==UnityWebRequest.Result.Success){created=true;loginUsername=data.username;registerMode=false;confirmPassword="";registerName="";}
            else{message="注册失败，请检查后端。若请求超时，请先尝试用该账号登录。";try{var error=JsonUtility.FromJson<ApiError>(request.downloadHandler.text);if(error?.error!=null)message=error.error.message;}catch(Exception){}}
        }
        loggingIn=false;
        if(created){yield return Login();if(currentUser==null)message="账号已注册。"+message;}
    }
    IEnumerator Login(){
        loggingIn=true;message="正在验证账号…";
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/inspector/login","POST")){
            request.uploadHandler=new UploadHandlerRaw(Encoding.UTF8.GetBytes(JsonUtility.ToJson(new LoginInput{username=loginUsername.Trim(),password=loginPassword})));request.downloadHandler=new DownloadHandlerBuffer();request.SetRequestHeader("Content-Type","application/json");request.timeout=10;
            yield return request.SendWebRequest();
            if(request.result==UnityWebRequest.Result.Success){try{var result=JsonUtility.FromJson<InspectorLogin>(request.downloadHandler.text);if(result.user==null||string.IsNullOrEmpty(result.token))throw new Exception();authToken=result.token;currentUser=result.user;loginPassword="";view.cullingMask=-1;message="登录成功，欢迎"+currentUser.displayName+"。";}catch(Exception){message="登录响应格式不正确。";}}
            else{Debug.LogWarning("Inspector login failed: HTTP "+request.responseCode+" "+request.error);message=request.responseCode==401?"账号或密码错误，请重新输入。":request.responseCode==403?"账号已停用，请联系管理员。":request.responseCode==429?"尝试次数过多，请一分钟后重试。":"无法登录，请检查后端服务和地址。";}
        }
        loggingIn=false;if(currentUser!=null)yield return Refresh();
    }
    IEnumerator Logout(){
        loading=true;
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/inspector/logout","POST")){request.downloadHandler=new DownloadHandlerBuffer();Authorize(request);request.timeout=5;yield return request.SendWebRequest();}
        ClearSession("已退出，请登录后继续巡检。");loading=false;
    }
    string PriorityLabel(string value){return value=="high"?"高":value=="medium"?"中":"低";}
    string StatusLabel(string value){return value=="resolved"?"已解决":value=="in_progress"?"处理中":"待处理";}
    string ErrorLabel(string value){switch(value){case "title is required":return "请填写问题标题。";case "priority must be low, medium or high":return "优先级不正确，请重新选择。";case "position.x/y/z are required":return "缺少标注坐标，请取消后重新标注。";case "could not persist issue":return "后端保存失败，请检查服务端磁盘和日志。";default:return "服务端未能保存问题，请检查后端服务后重试。";}}
    IEnumerator Refresh()
    {
        loading=true;message="正在加载已保存的问题标记…";
        using(var request=UnityWebRequest.Get(backendUrl.TrimEnd('/')+"/api/issues")){
            Authorize(request);
            request.timeout=10;yield return request.SendWebRequest();
            if(request.responseCode==401)ClearSession("登录已失效，请重新登录。");
            else if(request.result!=UnityWebRequest.Result.Success)message="无法连接后端，请确认 Go 服务和地址，再点击刷新。";
            else {try{var list=JsonUtility.FromJson<IssueList>("{\"items\":"+request.downloadHandler.text+"}");foreach(var issue in list.items)PutMarker(issue);message="已加载 "+list.items.Length+" 个问题标记。";}catch(Exception){message="后端返回的数据格式不正确。";}}
        }loading=false;
    }
    IEnumerator Submit()
    {
        submitting=true;message="正在提交问题，请稍候…";
        var input=new CreateIssue{title=title.Trim(),description=description.Trim(),priority=priorities[priority],position=new IssuePosition(hitPosition),deviceId=pendingDeviceId,area=pendingArea};
        using(var request=new UnityWebRequest(backendUrl.TrimEnd('/')+"/api/issues","POST")){
            request.uploadHandler=new UploadHandlerRaw(Encoding.UTF8.GetBytes(JsonUtility.ToJson(input)));request.downloadHandler=new DownloadHandlerBuffer();request.SetRequestHeader("Content-Type","application/json");request.timeout=10;
            Authorize(request);
            yield return request.SendWebRequest();
            if(request.responseCode==401)ClearSession("登录已失效，未提交标记已移除，请重新登录。");
            else if(request.result==UnityWebRequest.Result.Success){try{var issue=JsonUtility.FromJson<Issue>(request.downloadHandler.text);if(string.IsNullOrEmpty(issue.id)||issue.position==null||issue.reporter==null||issue.reporter.id!=currentUser.id)throw new Exception();PutMarker(issue);Destroy(pending);pending=null;formOpen=false;message="上报成功！请在管理端刷新查看。";}catch(Exception){message="响应格式异常，重试前请先到管理端核实是否已保存。";}}
            else{message="提交失败，请检查后端。重试前请先到管理端核实是否已保存。";try{var err=JsonUtility.FromJson<ApiError>(request.downloadHandler.text);if(err?.error!=null)message=ErrorLabel(err.error.message);}catch(Exception){}}
        }submitting=false;
    }
    IEnumerator CapturePreview(string path){yield return new WaitForSeconds(2);yield return new WaitForEndOfFrame();ScreenCapture.CaptureScreenshot(path);Debug.Log("INSPECTION_PREVIEW_REQUESTED");}
    IEnumerator SmokePreview(string suffix){string prefix=Environment.GetEnvironmentVariable("INSPECTION_CAPTURE_PREFIX");if(string.IsNullOrEmpty(prefix)||SystemInfo.graphicsDeviceType==UnityEngine.Rendering.GraphicsDeviceType.Null)yield break;yield return new WaitForSeconds(.4f);yield return new WaitForEndOfFrame();ScreenCapture.CaptureScreenshot(prefix+suffix+".png");yield return new WaitForSeconds(.5f);}
    // Opt-in integration check for CI/local validation; never runs in normal play.
    IEnumerator Smoke()
    {
        loginUsername=Environment.GetEnvironmentVariable("INSPECTION_SMOKE_USER")??"inspector01";
        string smokePassword=Environment.GetEnvironmentVariable("INSPECTION_SMOKE_PASSWORD")??"";loginPassword=smokePassword;
        if(Environment.GetEnvironmentVariable("INSPECTION_SMOKE_REGISTER")=="1"){registerName="集成检查巡检员";confirmPassword=loginPassword;yield return Register();}else yield return Login();
        if(currentUser==null){Debug.LogError("SMOKE_FAIL: inspector login failed: "+message);Application.Quit(1);yield break;}
        float deadline=Time.realtimeSinceStartup+20;
        while(loading&&Time.realtimeSinceStartup<deadline)yield return null;
        yield return new WaitForFixedUpdate();
        if(!Physics.Raycast(view.ViewportPointToRay(new Vector3(.5f,.5f)),out var hit,100f,1<<8)){
            Debug.LogError("SMOKE_FAIL: scene ray missed");Application.Quit(1);yield break;
        }
        SetHitLocation(hit);pending=MakeMarker(hitPosition,Color.yellow);formOpen=true;
        title="Unity runtime integration check";description="Created by opt-in smoke test";priority=2;
        yield return SmokePreview("-device");
        yield return Submit();
        if(formOpen||pending!=null){Debug.LogError("SMOKE_FAIL: "+message);Application.Quit(1);yield break;}
        yield return Refresh();
        if(markers.Count==0){Debug.LogError("SMOKE_FAIL: saved marker missing");Application.Quit(1);yield break;}
        if(Environment.GetEnvironmentVariable("INSPECTION_SMOKE_CHANGE_PASSWORD")=="1"){
            oldPassword=smokePassword;newPassword="changed-unity-test-password";newPasswordConfirmation=newPassword;passwordOpen=true;yield return SmokePreview("-password");yield return ChangePassword();
            if(currentUser!=null){Debug.LogError("SMOKE_FAIL: password change did not sign out");Application.Quit(1);yield break;}
            loginPassword="changed-unity-test-password";yield return Login();
            if(currentUser==null||markers.Count!=1){Debug.LogError("SMOKE_FAIL: cannot login with changed password");Application.Quit(1);yield break;}
            Debug.Log("INSPECTION_PASSWORD_PASS");
        }
        Debug.Log("INSPECTION_SMOKE_PASS: physics ray -> UnityWebRequest -> Go save -> reload markers");
        Application.Quit(0);
    }
}
