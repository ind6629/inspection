using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
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
        var report=BuildPipeline.BuildPlayer(new BuildPlayerOptions{scenes=new[]{"Assets/Scenes/Inspection.unity"},locationPathName="BuildManaged/Inspection.exe",target=BuildTarget.StandaloneWindows64,options=BuildOptions.None});
        if(report.summary.result!=UnityEditor.Build.Reporting.BuildResult.Succeeded)throw new System.Exception("Build failed");
    }
}
