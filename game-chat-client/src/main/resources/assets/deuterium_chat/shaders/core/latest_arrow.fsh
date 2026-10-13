#version 150
in vec2 texCoord0;
in vec4 vertexColor;
out vec4 fragColor;

float edgeDistance(vec2 p,vec2 a,vec2 b){
    vec2 edge=b-a;
    return length(p-a-edge*clamp(dot(p-a,edge)/dot(edge,edge),0.0,1.0));
}
void main(){
    // ShaderTexture supplies bottom-up UVs; use screen-down coordinates here.
    vec2 p=vec2(texCoord0.x,1.0-texCoord0.y)-.5;
    vec2 a=vec2(-.30,-.17),b=vec2(.30,-.17),c=vec2(0.0,.23);
    float d=min(edgeDistance(p,a,b),min(edgeDistance(p,b,c),edgeDistance(p,c,a)));
    if(p.y>=-.17 && abs(p.x)*(4.0/3.0)+p.y<=.23)d=-d;
    d-=.025;
    float aa=max(fwidth(d),.0001);
    fragColor=vec4(vertexColor.rgb,vertexColor.a*(1.0-smoothstep(-aa*.5,aa*.5,d)));
}
