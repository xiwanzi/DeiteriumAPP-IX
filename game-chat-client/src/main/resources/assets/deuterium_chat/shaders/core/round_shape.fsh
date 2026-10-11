#version 150
uniform vec2 ShapeSize;
uniform float Radius;
uniform float Stroke;
uniform vec4 FillColor;
uniform vec4 BorderColor;
in vec2 texCoord0;
in vec4 vertexColor;
out vec4 fragColor;
void main(){
    vec2 q=abs((texCoord0-.5)*ShapeSize)-(ShapeSize*.5-vec2(Radius));
    float d=length(max(q,0.0))+min(max(q.x,q.y),0.0)-Radius;
    float aa=max(fwidth(d),.0001);
    float outer=1.0-smoothstep(-aa*.5,aa*.5,d);
    float inner=Stroke<=0.0?outer:1.0-smoothstep(-aa*.5,aa*.5,d+Stroke);
    float fa=inner*FillColor.a,ba=(outer-inner)*BorderColor.a;
    float a=fa+ba;
    fragColor=vec4((FillColor.rgb*fa+BorderColor.rgb*ba)/max(a,.00001),a)*vertexColor;
}
