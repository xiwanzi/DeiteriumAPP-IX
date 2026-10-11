#version 150
uniform sampler2D Sampler0;
in vec2 texCoord0;
in vec4 vertexColor;
out vec4 fragColor;
void main(){fragColor=vec4(vertexColor.rgb,texture(Sampler0,texCoord0).a*vertexColor.a);}
