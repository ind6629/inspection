using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEditor.Build;
public static class ProjectSetup
{
    [MenuItem("Inspection/Create or open demo scene")]
    public static void CreateScene()
    {
        System.IO.Directory.CreateDirectory("Assets/Scenes");
        System.IO.Directory.CreateDirectory("Assets/Resources");
        var shader=Shader.Find("Inspection/SiteSurface");
        if(shader==null)throw new System.Exception("SiteSurface shader not imported");
        if(AssetDatabase.LoadAssetAtPath<Material>("Assets/Resources/SiteSurface.mat")==null)
            AssetDatabase.CreateAsset(new Material(shader),"Assets/Resources/SiteSurface.mat");
        var scene=EditorSceneManager.NewScene(NewSceneSetup.EmptyScene,NewSceneMode.Single);
        new GameObject("InspectionApp").AddComponent<InspectionApp>();
        EditorSceneManager.SaveScene(scene,"Assets/Scenes/Inspection.unity");
        EditorBuildSettings.scenes=new[]{new EditorBuildSettingsScene("Assets/Scenes/Inspection.unity",true)};
        PlayerSettings.productName="虚拟现场巡检";
        PlayerSettings.companyName="InspectionDemo";
        PlayerSettings.defaultScreenWidth=1280;PlayerSettings.defaultScreenHeight=800;
        PlayerSettings.runInBackground=true;
        PlayerSettings.fullScreenMode=FullScreenMode.Windowed;
        AssetDatabase.SaveAssets();
        Debug.Log("INSPECTION_PROJECT_READY");
    }
    public static void BuildWindows()
    {
        CreateScene();
        var report=BuildPipeline.BuildPlayer(new BuildPlayerOptions{scenes=new[]{"Assets/Scenes/Inspection.unity"},locationPathName="BuildLocationPassword/Inspection.exe",target=BuildTarget.StandaloneWindows64,options=BuildOptions.None});
        if(report.summary.result!=UnityEditor.Build.Reporting.BuildResult.Succeeded)throw new System.Exception("Build failed");
    }
    public static void BuildAndroid()
    {
        CreateScene();
        var endpoint=System.Environment.GetEnvironmentVariable("INSPECTION_ANDROID_BACKEND");
        if(!string.IsNullOrWhiteSpace(endpoint)){System.IO.File.WriteAllText("Assets/Resources/AndroidBackend.txt",endpoint.Trim());AssetDatabase.ImportAsset("Assets/Resources/AndroidBackend.txt");}
        PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android,"com.inspectiondemo.virtualsite");
        PlayerSettings.bundleVersion="1.0.0";PlayerSettings.Android.bundleVersionCode=1;
        PlayerSettings.SetScriptingBackend(NamedBuildTarget.Android,ScriptingImplementation.IL2CPP);
        PlayerSettings.Android.targetArchitectures=AndroidArchitecture.ARM64;
        PlayerSettings.Android.minSdkVersion=AndroidSdkVersions.AndroidApiLevel26;
        PlayerSettings.Android.targetSdkVersion=(AndroidSdkVersions)36;
        PlayerSettings.Android.useCustomKeystore=false;
        PlayerSettings.Android.forceInternetPermission=true;
        PlayerSettings.insecureHttpOption=InsecureHttpOption.AlwaysAllowed;
        PlayerSettings.defaultInterfaceOrientation=UIOrientation.LandscapeLeft;
        PlayerSettings.SetUseDefaultGraphicsAPIs(BuildTarget.Android,false);
        PlayerSettings.SetGraphicsAPIs(BuildTarget.Android,new[]{UnityEngine.Rendering.GraphicsDeviceType.OpenGLES3});
        EditorUserBuildSettings.buildAppBundle=false;
        TouchNavigation.Verify();
        var report=BuildPipeline.BuildPlayer(new BuildPlayerOptions{scenes=new[]{"Assets/Scenes/Inspection.unity"},locationPathName="../android/Inspection-Android.apk",target=BuildTarget.Android,options=BuildOptions.CompressWithLz4});
        if(report.summary.result!=UnityEditor.Build.Reporting.BuildResult.Succeeded)throw new System.Exception("Android build failed");
        Debug.Log("INSPECTION_ANDROID_BUILD_PASS");
    }
}
