using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
public static class ProjectSetup
{
    [MenuItem("Inspection/Create or open demo scene")]
    public static void CreateScene()
    {
        System.IO.Directory.CreateDirectory("Assets/Scenes");
        var scene=EditorSceneManager.NewScene(NewSceneSetup.EmptyScene,NewSceneMode.Single);
        new GameObject("InspectionApp").AddComponent<InspectionApp>();
        EditorSceneManager.SaveScene(scene,"Assets/Scenes/Inspection.unity");
        EditorBuildSettings.scenes=new[]{new EditorBuildSettingsScene("Assets/Scenes/Inspection.unity",true)};
        PlayerSettings.productName="Virtual Site Inspection";
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
        var report=BuildPipeline.BuildPlayer(new BuildPlayerOptions{scenes=new[]{"Assets/Scenes/Inspection.unity"},locationPathName="Build/Inspection.exe",target=BuildTarget.StandaloneWindows64,options=BuildOptions.None});
        if(report.summary.result!=UnityEditor.Build.Reporting.BuildResult.Succeeded)throw new System.Exception("Build failed");
    }
}
