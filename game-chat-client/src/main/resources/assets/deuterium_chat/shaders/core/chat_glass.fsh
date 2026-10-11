#version 150
uniform sampler2D Sampler0;
uniform sampler2D Grain;
uniform vec2 ScreenSize;
uniform vec2 TexelSize;
uniform vec2 BlurAxis;
uniform float FullPanel;
uniform float Reveal;
uniform float BlurRadius;
in vec2 texCoord0;
in vec4 vertexColor;
out vec4 fragColor;
void main() {
    // Separable Gaussian, sigma ~14 physical pixels, horizontal at quarter resolution.
    vec2 d=TexelSize*BlurAxis*(BlurRadius/6.0);
    vec3 c=texture(Sampler0,texCoord0).rgb*0.26656;
    c+=(texture(Sampler0,texCoord0+d).rgb+texture(Sampler0,texCoord0-d).rgb)*0.21344;
    c+=(texture(Sampler0,texCoord0+2.0*d).rgb+texture(Sampler0,texCoord0-2.0*d).rgb)*0.10959;
    c+=(texture(Sampler0,texCoord0+3.0*d).rgb+texture(Sampler0,texCoord0-3.0*d).rgb)*0.03607;
    c+=(texture(Sampler0,texCoord0+4.0*d).rgb+texture(Sampler0,texCoord0-4.0*d).rgb)*0.00761;
    if(FullPanel<0.0){fragColor=vec4(c,1.0);return;}
    float x=texCoord0.x;
    float blurMask=1.0,shade=.65;
    vec3 tint=vec3(14,20,29)/255.0;
    if(abs(FullPanel-2.0)<.1){shade=.78;tint=vec3(22,29,32)/255.0;}
    if(FullPanel>2.5)shade=0.0;
    if(abs(FullPanel-1.0)<.1){
        blurMask=x<.24?1.0:x<.42?mix(1.0,.733,(x-.24)/.18):x<.58?mix(.733,.267,(x-.42)/.16):x<.76?mix(.267,0.0,(x-.58)/.18):0.0;
        shade=x<.18?mix(.988,.957,x/.18):x<.34?mix(.957,.910,(x-.18)/.16):x<.48?mix(.910,.733,(x-.34)/.14):x<.61?mix(.733,.408,(x-.48)/.13):x<.75?mix(.408,.110,(x-.61)/.14):x<.90?mix(.110,0.0,(x-.75)/.15):0.0;
        vec3 a0=vec3(16,20,23)/255.0,a1=vec3(17,22,25)/255.0,a2=vec3(19,25,27)/255.0,a3=vec3(20,26,29)/255.0;
        tint=x<.18?mix(a0,a1,x/.18):x<.34?mix(a1,a2,(x-.18)/.16):x<.48?a2:x<.61?mix(a2,a3,(x-.48)/.13):a3;
    }
    // Blur and shade masks compose independently, as backdrop-filter + shade do in Web.
    float a=blurMask+shade-blurMask*shade;
    vec3 premultiplied=c*blurMask*(1.0-shade)+tint*shade;
    if(abs(FullPanel-1.0)<.1){
        vec4 grain=texture(Grain,fract(vec2(texCoord0.x,1.0-texCoord0.y)*ScreenSize/vec2(142.0,66.0)));
        float ga=grain.a*.022*max(0.0,1.0-x/.65);
        premultiplied=premultiplied*(1.0-ga)+grain.rgb*ga;
        a=a+ga-a*ga;
    }
    fragColor=vec4(premultiplied/max(a,.0001),a*Reveal)*vertexColor;
}
