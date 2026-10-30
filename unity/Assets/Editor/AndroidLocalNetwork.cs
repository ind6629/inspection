#if UNITY_ANDROID
using System.IO;
using System.Xml;
using UnityEditor.Android;

// This demo intentionally talks to a local HTTP Go server over the same Wi-Fi.
public sealed class AndroidLocalNetwork : IPostGenerateGradleAndroidProject
{
    public int callbackOrder => 0;
    public void OnPostGenerateGradleAndroidProject(string path)
    {
        string manifestPath=Path.Combine(path,"src/main/AndroidManifest.xml");
        var document=new XmlDocument();document.Load(manifestPath);
        var application=document.SelectSingleNode("/manifest/application") as XmlElement;
        if(application==null)throw new System.Exception("Android application manifest missing");
        application.SetAttribute("usesCleartextTraffic","http://schemas.android.com/apk/res/android","true");
        document.Save(manifestPath);
    }
}
#endif
