using System.Collections.Generic;
using UnityEngine;

// Positions and deltas use the same safe-area UI coordinates as IMGUI.
public sealed class TouchNavigation
{
    sealed class Gesture { public Vector2 start;public float time,distance;public int role; }
    readonly Dictionary<int,Gesture> fingers=new Dictionary<int,Gesture>();
    public Vector2 Move {get;private set;}
    public Vector2 Look {get;private set;}
    public void BeginFrame(){Move=Vector2.zero;Look=Vector2.zero;}
    public void Clear(){fingers.Clear();BeginFrame();}
    public bool Process(int id,TouchPhase phase,Vector2 point,Vector2 delta,float now,bool blocked,bool onPad,Vector2 padCenter)
    {
        if(phase==TouchPhase.Began)fingers[id]=new Gesture{start=point,time=now,role=blocked?0:onPad?1:2};
        if(!fingers.TryGetValue(id,out var gesture))return false;
        gesture.distance=Mathf.Max(gesture.distance,Vector2.Distance(point,gesture.start));
        if(phase==TouchPhase.Canceled){fingers.Remove(id);return false;}
        if(phase==TouchPhase.Ended){fingers.Remove(id);return gesture.role==2&&gesture.distance<=12&&now-gesture.time<.45f;}
        if(gesture.role==1)Move=Vector2.ClampMagnitude(new Vector2((point.x-padCenter.x)/80,(padCenter.y-point.y)/80),1);
        if(gesture.role==2&&gesture.distance>12&&phase==TouchPhase.Moved)Look+=delta;
        return false;
    }
    public static void Verify()
    {
        var n=new TouchNavigation();var p=Vector2.zero;
        n.BeginFrame();if(n.Process(1,TouchPhase.Began,p,p,0,false,false,p))throw new System.Exception("tap fired on begin");
        if(!n.Process(1,TouchPhase.Ended,p,p,.1f,false,false,p))throw new System.Exception("tap not recognized");
        if(n.Process(1,TouchPhase.Ended,p,p,.2f,false,false,p))throw new System.Exception("duplicate tap");
        n.Process(2,TouchPhase.Began,p,p,0,true,false,p);if(n.Process(2,TouchPhase.Ended,p,p,.1f,false,false,p))throw new System.Exception("UI tap escaped");
        n.Process(3,TouchPhase.Began,p,p,0,false,false,p);n.Process(3,TouchPhase.Moved,new Vector2(30,0),new Vector2(30,0),.1f,false,false,p);if(n.Look.x==0||n.Process(3,TouchPhase.Ended,new Vector2(30,0),p,.2f,false,false,p))throw new System.Exception("drag turned into tap");
        n.BeginFrame();n.Process(4,TouchPhase.Began,new Vector2(0,-80),p,0,false,true,p);if(n.Move.y<.9f||n.Process(4,TouchPhase.Ended,p,p,.1f,false,false,p))throw new System.Exception("movement pad incorrect");
        n.Process(5,TouchPhase.Began,p,p,0,false,false,p);if(n.Process(5,TouchPhase.Ended,p,p,1,false,false,p))throw new System.Exception("long press became tap");
        Debug.Log("TOUCH_NAVIGATION_PASS");
    }
}
