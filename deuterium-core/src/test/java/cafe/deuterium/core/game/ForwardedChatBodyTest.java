package cafe.deuterium.core.game;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
class ForwardedChatBodyTest {
 @Test void removesInstalledFormatterEnvelopeOnly(){
  assertEquals("今天出发吗",ForwardedChatBody.fromTrChat("[Amiya] xiwanzi: 今天出发吗","xiwanzi"));
  assertEquals("[App] xiwanzi: 用户自己输入的文字",ForwardedChatBody.fromTrChat("[world] xiwanzi: [App] xiwanzi: 用户自己输入的文字","xiwanzi"));
  assertEquals("hello",ForwardedChatBody.fromTrChat("xiwanzi: hello","xiwanzi"));
  assertEquals("[world] Other: hello",ForwardedChatBody.fromTrChat("[world] Other: hello","xiwanzi"));
 }
}
