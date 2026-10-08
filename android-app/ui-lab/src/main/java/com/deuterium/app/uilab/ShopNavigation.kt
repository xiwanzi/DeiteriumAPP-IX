package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.lazy.staggeredgrid.*
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.foundation.shape.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch

data class ShopCategory(val id:String,val name:String,val parentId:String="",val order:Int=0,val image:String?=null)
data class ShopBanner(val id:String,val title:String,val image:String,val order:Int=0)
const val NEW_PRODUCT_BADGE="新品推荐"
fun shopSearchRoute(query:String)="search:Shop:"+android.net.Uri.encode(query)
fun shopRootCategories(categories:List<ShopCategory>)=categories.filter{it.parentId.isBlank()||categories.none{p->p.id==it.parentId}}.distinctBy{it.name}

fun shopGroupChildren(categories:List<ShopCategory>,name:String):List<ShopCategory> {
    val parents=categories.filter{it.name==name&&it.parentId.isBlank()}.map{it.id}.toSet()
    return categories.filter{it.parentId in parents}.distinctBy{it.name}
}

@Composable
fun ShopPage(state:LabState,onProduct:(String)->Unit,topInset:Dp,query:String,onOpen:(String)->Unit={}) {
    var category by rememberSaveable { mutableStateOf("") }
    val network=state.commerce.network
    val page=network?.storePage
    val scope=rememberCoroutineScope()
    val list=rememberLazyStaggeredGridState()
    val motion=LocalMotion.current
    val categories=network?.categoryTree?.toList().orEmpty()
    val roots=remember(categories){shopRootCategories(categories)}
    val selectedGroup=roots.firstOrNull{it.id==category}?.name.orEmpty()
    val children=remember(selectedGroup,categories){shopGroupChildren(categories,selectedGroup)}
    LaunchedEffect(selectedGroup,query){network?.refreshStore(query,categoryGroup=selectedGroup)}
    val allProducts=page?.items?.toList().orEmpty()
    val showcase=remember(allProducts,category){if(category.isBlank())allProducts.take(6)else emptyList()}
    val featuredIds=remember(showcase){showcase.map{it.id}.toSet()}
    val products=if(category.isBlank())allProducts.filterNot{it.id in featuredIds}else allProducts
    CatalogAutoLoad(page,list,products)
    Column(Modifier.fillMaxSize()) {
        Column {
            Spacer(Modifier.height(topInset))
            Box(Modifier.fillMaxWidth().height(8.dp).semantics{contentDescription="回到商城顶部"}.clickable{scope.launch{if(motion)list.animateScrollToItem(0)else list.scrollToItem(0)}})
            Row(Modifier.padding(start=12.dp,end=4.dp),verticalAlignment=Alignment.CenterVertically) {
                Row(Modifier.weight(1f).height(44.dp).clip(RoundedCornerShape(14.dp)).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.58f)).clickable{onOpen("search:Shop")}.padding(horizontal=15.dp),verticalAlignment=Alignment.CenterVertically) {
                    Icon(Icons.Outlined.Search,null,Modifier.size(21.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)
                    Text("搜索商品、品牌或分类",Modifier.padding(start=10.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
                }
                ShoppingBagButton(state.cart.values.sum()){onOpen("bag")}
            }
            Row(Modifier.fillMaxWidth().padding(top=2.dp,bottom=2.dp),verticalAlignment=Alignment.CenterVertically) {
                LazyRow(Modifier.weight(1f),contentPadding=PaddingValues(start=8.dp,end=0.dp),horizontalArrangement=Arrangement.spacedBy(2.dp)) {
                    item{ShopChannel("首页",category.isBlank()){category="";scope.launch{list.scrollToItem(0)}}}
                    items(roots,key={it.id}){c->ShopChannel(c.name,category==c.id){category=c.id;scope.launch{list.scrollToItem(0)}}}
                }
                IconButton({onOpen("shop-categories:$category")}){Icon(Icons.Outlined.Menu,"更多分类")}
            }
        }
        LazyVerticalStaggeredGrid(columns=StaggeredGridCells.Fixed(2),modifier=Modifier.weight(1f),state=list,contentPadding=PaddingValues(start=6.dp,end=6.dp,top=4.dp,bottom=115.dp),verticalItemSpacing=7.dp,horizontalArrangement=Arrangement.spacedBy(7.dp)) {
            if(category.isBlank()&&showcase.isNotEmpty()) {
                item(key="shop-showcase",span=StaggeredGridItemSpan.FullLine){
                    LazyRow(horizontalArrangement=Arrangement.spacedBy(12.dp)) {
                        items(showcase,key={it.id}){product->ProductPoster(product,Modifier.width(308.dp).height(405.dp)){onProduct(product.id)}}
                    }
                }
            } else if(children.isNotEmpty()) {
                item(span=StaggeredGridItemSpan.FullLine){ShopCategoryTiles(children.take(if(children.size>15)14 else 15),columns=5,onSelect={onOpen(shopSearchRoute(it.name))},more=if(children.size>15){{onOpen("shop-categories:$category")}}else null)}
            }
            shopProducts(products,onProduct){product->
                if(network==null||network.cartBusy)false else network.setCart(product.id,((state.cart[product.id] ?: 0)+1).coerceAtMost(product.limit))
            }
            catalogStatus(page,products.isEmpty()&&(category.isNotBlank()||showcase.isEmpty()),"暂无上架商品",scope)
        }
    }
}

@Composable
internal fun CatalogAutoLoad(page:CatalogPager<ShopProduct>?,list:LazyStaggeredGridState,products:List<ShopProduct>,active:Boolean=true) {
    val scope=rememberCoroutineScope();val images=AppImages.get(LocalContext.current)
    val indices=remember(products){products.mapIndexed{index,product->product.id to index}.toMap()}
    val last by remember(indices){derivedStateOf{list.layoutInfo.visibleItemsInfo.mapNotNull{indices[it.key]}.maxOrNull() ?: -1}}
    val more=active&&products.isNotEmpty()&&last>=products.size-12&&page?.hasMore==true&&!page.loading&&page.error==null
    LaunchedEffect(more){if(more)scope.launch{page?.more()}}
    LaunchedEffect(products,images.epoch,active){if(active)snapshotFlow{last}.distinctUntilChanged().collectLatest{index->images.prefetch(products.drop((index+1).coerceAtLeast(0)).take(24).mapNotNull{it.photos.firstOrNull()})}}
}

@Composable
private fun ShopChannel(label:String,selected:Boolean,onClick:()->Unit) {
    Column(Modifier.height(44.dp).clickable(onClick=onClick).padding(horizontal=9.dp),horizontalAlignment=Alignment.CenterHorizontally,verticalArrangement=Arrangement.Center) {
        Text(label,fontSize=16.sp,fontWeight=if(selected)FontWeight.Bold else FontWeight.Medium,color=if(selected)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface)
        Box(Modifier.padding(top=4.dp).size(20.dp,3.dp).clip(CircleShape).background(if(selected)MaterialTheme.colorScheme.primary else Color.Transparent))
    }
}

internal fun LazyStaggeredGridScope.shopProducts(products:List<ShopProduct>,onProduct:(String)->Unit,onAdd:suspend(ShopProduct)->Boolean) {
    items(products,key={it.id}){product->ShopProductCard(product,{onProduct(product.id)}){onAdd(product)}}
}

@Composable
private fun ShopProductCard(product:ShopProduct,onClick:()->Unit,onAdd:suspend()->Boolean) {
    var adding by remember(product.id){mutableStateOf(false)}
    var added by remember(product.id){mutableStateOf(false)}
    val scope=rememberCoroutineScope()
    val offer=rememberShopOffer(product)
    var couponDetails by remember(product.id){mutableStateOf(false)}
    val promotion=product.originalPrice>product.price
    val promoColor=ShopDiscountPink
    LaunchedEffect(added){if(added){delay(1000);added=false}}
    Surface(onClick=onClick,modifier=Modifier.fillMaxWidth(),shape=RoundedCornerShape(8.dp),color=MaterialTheme.colorScheme.surface) {
        Column {
            Box {
                ShopSubjectImage(product.photos.firstOrNull(),product.name,Modifier.fillMaxWidth().aspectRatio(1f).clip(RoundedCornerShape(8.dp)))
            }
            Column(Modifier.padding(start=8.dp,end=5.dp,top=6.dp,bottom=3.dp),verticalArrangement=Arrangement.spacedBy(2.dp)) {
                Text(product.name,fontSize=14.sp,lineHeight=19.sp,fontWeight=FontWeight.SemiBold,maxLines=1,overflow=TextOverflow.Ellipsis)
                if(promotion)Row(verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(4.dp)) {
                    Text("促销",Modifier.clip(RoundedCornerShape(3.dp)).background(promoColor.copy(alpha=.10f)).padding(horizontal=4.dp,vertical=1.dp),fontSize=10.sp,lineHeight=14.sp,color=promoColor)
                    Text("直降 ${shopMoney(product.originalPrice-product.price)}",fontSize=11.sp,lineHeight=15.sp,color=promoColor)
                }
                if(offer!=null)Row(Modifier.fillMaxWidth().clickable{couponDetails=true}.padding(vertical=2.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(4.dp)) {
                    Text("券",Modifier.border(1.dp,promoColor.copy(alpha=.5f),RoundedCornerShape(3.dp)).padding(horizontal=3.dp),fontSize=10.sp,lineHeight=14.sp,color=promoColor)
                    Text("${offer.couponName} · ${offer.condition}",Modifier.weight(1f),fontSize=11.sp,lineHeight=15.sp,color=promoColor,maxLines=2,overflow=TextOverflow.Ellipsis)
                }
                if(!promotion&&offer==null)Text(product.category,fontSize=11.sp,lineHeight=15.sp,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
                Row(verticalAlignment=Alignment.CenterVertically) {
                    CompactShopPrice(offer?.finalPrice ?: product.price,Modifier.weight(1f),if(offer!=null)"券后"else if(promotion)"促销价"else"")
                    IconButton({if(!adding)scope.launch{adding=true;try{added=onAdd()}finally{adding=false}}},Modifier.size(36.dp),enabled=!adding&&product.stock>0){
                        Box(Modifier.size(23.dp).background(MaterialTheme.colorScheme.primaryContainer,CircleShape),contentAlignment=Alignment.Center){Icon(if(added)Icons.Outlined.Check else Icons.Outlined.Add,"加入购物袋：${product.name}",Modifier.size(17.dp),tint=MaterialTheme.colorScheme.primary)}
                    }
                }
                if(product.originalPrice>(offer?.finalPrice ?: product.price))Text(shopMoney(product.originalPrice),lineHeight=14.sp,fontSize=11.sp,textDecoration=TextDecoration.LineThrough,color=MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
    if(couponDetails&&offer!=null)IosDialog({couponDetails=false},{Text("已为你优选 ${offer.couponName}")},{Column(verticalArrangement=Arrangement.spacedBy(10.dp)){
        Text(offer.condition)
        Text(if(offer.stack)"可与商品促销同享"else"与商品促销自动择优")
        Text("单件券后 ${shopMoney(offer.finalPrice)}",fontWeight=FontWeight.SemiBold,color=shopPriceColor(true))
        Text("按当前可用券比较后选择。单件金额为预估，优惠有效性与实付以结算为准。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
    }},{PlainButton({couponDetails=false}){Text("知道了")}})

}

@Composable
private fun CompactShopPrice(amount:Long,modifier:Modifier=Modifier,label:String="") {
    val measurer=rememberTextMeasurer();val density=LocalDensity.current
    val primary=shopPriceColor(label.isNotEmpty());val muted=primary
    val style=MaterialTheme.typography.bodyMedium.copy(lineHeight=24.sp,fontFeatureSettings="tnum")
    fun text(scale:Float)=buildAnnotatedString {
        withStyle(SpanStyle(fontSize=(20*scale).sp,fontWeight=FontWeight.SemiBold,color=primary)){append(shopMoney(amount))}
        if(label.isNotEmpty())withStyle(SpanStyle(fontSize=(10*scale).sp,color=muted)){append(" $label")}
    }
    BoxWithConstraints(modifier){
        val width=with(density){maxWidth.toPx()};val measured=measurer.measure(text(1f),style,maxLines=1,softWrap=false).size.width
        val scale=if(measured>width&&measured>0)width/measured else 1f
        Text(text(scale),style=style,maxLines=1,softWrap=false)
    }
}

internal fun LazyStaggeredGridScope.catalogStatus(page:CatalogPager<ShopProduct>?,empty:Boolean,emptyText:String,scope:kotlinx.coroutines.CoroutineScope) {
    if(page?.loading==true)item(span=StaggeredGridItemSpan.FullLine){Text("正在加载商品…",Modifier.padding(24.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)}
    else if(page?.error!=null)item(span=StaggeredGridItemSpan.FullLine){CatalogMessage(page.error.orEmpty()){scope.launch{page.retry()}}}
    else if(empty)item(span=StaggeredGridItemSpan.FullLine){Text(emptyText,Modifier.fillMaxWidth().padding(32.dp),textAlign=TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)}
    else if(page?.hasMore==true)item(span=StaggeredGridItemSpan.FullLine){Box(Modifier.fillMaxWidth(),contentAlignment=Alignment.Center){PlainButton({scope.launch{page.more()}}){Text("加载更多")}}}
}

@Composable
private fun CatalogMessage(text:String,retry:()->Unit) {Row(Modifier.padding(horizontal=24.dp),verticalAlignment=Alignment.CenterVertically){Text(text,Modifier.weight(1f),color=MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.bodyMedium);PlainButton(retry){Text("重试")}}}

@Composable
private fun ShopCategoryTiles(categories:List<ShopCategory>,columns:Int,onSelect:(ShopCategory)->Unit,more:(()->Unit)?=null,directory:Boolean=false) {
    val cells:List<ShopCategory?> = categories+if(more!=null)listOf(null)else emptyList()
    Column(Modifier.padding(horizontal=10.dp,vertical=4.dp),verticalArrangement=Arrangement.spacedBy(if(directory)14.dp else 10.dp)) {
        cells.chunked(columns).forEach{row->Row(horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            row.forEach{c->Column(Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).clickable{if(c==null)more?.invoke()else onSelect(c)}.padding(vertical=4.dp),horizontalAlignment=Alignment.CenterHorizontally) {
                val image=if(c==null)null else c.image ?: ShopCatalog.firstOrNull{it.categoryId==c.id}?.photos?.firstOrNull()
                if(image!=null)ShopSubjectImage(image,c?.name.orEmpty(),Modifier.widthIn(max=72.dp).fillMaxWidth().aspectRatio(1f).clip(RoundedCornerShape(10.dp)))
                else Box(Modifier.widthIn(max=72.dp).fillMaxWidth().aspectRatio(1f).background(MaterialTheme.colorScheme.surface,CircleShape),contentAlignment=Alignment.Center){Icon(if(c==null)Icons.Outlined.MoreHoriz else Icons.Outlined.Widgets,null,Modifier.size(25.dp),tint=MaterialTheme.colorScheme.primary)}
                Text(c?.name ?: "更多",Modifier.padding(top=6.dp),fontSize=12.sp,lineHeight=16.sp,maxLines=2,minLines=if(directory)1 else 2,overflow=TextOverflow.Ellipsis,textAlign=TextAlign.Center)
            }}
            repeat(columns-row.size){Spacer(Modifier.weight(1f))}
        }}
    }
}

internal data class ShopDirectorySection(val name:String,val categories:List<ShopCategory>)

internal fun shopDirectorySections(root:String,categories:List<ShopCategory>):List<ShopDirectorySection> {
    if(categories.isEmpty())return emptyList()
    val grouped=categories.groupBy { c->
        when(root) {
            "家具家居"->when {
                listOf("床","座椅","沙发","桌台").any{c.name.contains(it)}->"起居家具"
                listOf("厨房","卫浴").any{c.name.contains(it)}->"厨房卫浴"
                else->"收纳陈设"
            }
            "灯具照明"->if(c.name.contains("庭院"))"庭院照明"else"室内照明"
            else->root
        }
    }
    return grouped.map{ShopDirectorySection(it.key,it.value)}
}

@Composable
private fun DirectoryTab(label:String,selected:Boolean,onClick:()->Unit) {
    val primary=MaterialTheme.colorScheme.primary
    Box(Modifier.heightIn(min=32.dp).clip(RoundedCornerShape(7.dp)).background(if(selected)primary.copy(alpha=.07f)else MaterialTheme.colorScheme.surface).border(1.dp,if(selected)primary.copy(alpha=.65f)else Color.Transparent,RoundedCornerShape(7.dp)).clickable(onClick=onClick).padding(horizontal=12.dp,vertical=7.dp),contentAlignment=Alignment.Center){Text(label,fontSize=12.sp,fontWeight=if(selected)FontWeight.SemiBold else FontWeight.Normal,color=if(selected)primary else MaterialTheme.colorScheme.onSurface,maxLines=1)}
}

@Composable
fun ShopCategoriesPage(state:LabState,initial:String,onClose:()->Unit,onOpen:(String)->Unit) {
    val network=state.commerce.network
    LaunchedEffect(Unit){network?.ensureMetadata()}
    val categories=network?.categoryTree?.toList().orEmpty();val roots=remember(categories){shopRootCategories(categories)}
    var selected by rememberSaveable {mutableStateOf(initial)}
    val current=roots.firstOrNull{it.id==selected} ?: roots.firstOrNull()
    val children=shopGroupChildren(categories,current?.name.orEmpty())
    val sections=remember(current,children){shopDirectorySections(current?.name.orEmpty(),if(children.isEmpty())listOfNotNull(current)else children)}
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).statusBarsPadding().navigationBarsPadding()) {
        Row(Modifier.fillMaxWidth().height(48.dp),verticalAlignment=Alignment.CenterVertically){IconButton(onClose){Icon(Icons.Outlined.ChevronLeft,"返回")};Text("分类",Modifier.weight(1f),textAlign=TextAlign.Center,fontSize=20.sp,fontWeight=FontWeight.SemiBold);IconButton({onOpen("search:Shop")}){Icon(Icons.Outlined.Search,"搜索商品")}}
        if(roots.isEmpty()){val scope=rememberCoroutineScope();CatalogMessage(network?.shopError ?: "暂无商品分类"){scope.launch{network?.ensureMetadata()}}}
        Row(Modifier.weight(1f)) {
            LazyColumn(Modifier.width(104.dp).fillMaxHeight(),contentPadding=PaddingValues(top=6.dp,bottom=12.dp)) {
                items(roots,key={it.id}){c->Surface(onClick={selected=c.id},color=MaterialTheme.colorScheme.background){Text(c.name,Modifier.fillMaxWidth().padding(horizontal=8.dp,vertical=19.dp),color=if(current?.id==c.id)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,fontSize=14.sp,fontWeight=if(current?.id==c.id)FontWeight.Bold else FontWeight.Normal,textAlign=TextAlign.Center)}}
            }
            key(current?.id){
                val list=rememberLazyListState();val scope=rememberCoroutineScope();val motion=LocalMotion.current
                var expanded by remember{mutableStateOf(false)}
                var active by remember{mutableIntStateOf(0)}
                LaunchedEffect(list){snapshotFlow{list.firstVisibleItemIndex}.distinctUntilChanged().collect{if(list.isScrollInProgress)active=it.coerceAtMost((sections.size-1).coerceAtLeast(0))}}
                fun selectSection(index:Int){active=index;expanded=false;scope.launch{if(motion)list.animateScrollToItem(index)else list.scrollToItem(index);active=index}}
                Column(Modifier.weight(1f).fillMaxHeight().padding(end=8.dp)) {
                    if(sections.size>1) {
                        Row(Modifier.fillMaxWidth().height(48.dp),verticalAlignment=Alignment.CenterVertically){
                            LazyRow(Modifier.weight(1f),horizontalArrangement=Arrangement.spacedBy(6.dp)){items(sections.size){index->DirectoryTab(sections[index].name,index==active){selectSection(index)}}}
                            IconButton({expanded=!expanded},Modifier.size(32.dp)){Icon(if(expanded)Icons.Outlined.ExpandLess else Icons.Outlined.ExpandMore,"展开全部分组",Modifier.size(20.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}
                        }
                        if(expanded)FlowRow(Modifier.padding(bottom=8.dp),horizontalArrangement=Arrangement.spacedBy(6.dp),verticalArrangement=Arrangement.spacedBy(6.dp)){sections.forEachIndexed{index,section->DirectoryTab(section.name,index==active){selectSection(index)}}}
                    } else Spacer(Modifier.height(8.dp))
                    LazyColumn(Modifier.weight(1f).fillMaxWidth().clip(RoundedCornerShape(topStart=16.dp,topEnd=16.dp)).background(MaterialTheme.colorScheme.surface),state=list,contentPadding=PaddingValues(top=6.dp,bottom=24.dp),verticalArrangement=Arrangement.spacedBy(8.dp)) {
                        items(sections,key={it.name}){section->Column {
                            Text(section.name,Modifier.padding(start=14.dp,end=12.dp,top=10.dp,bottom=4.dp),fontSize=17.sp,lineHeight=23.sp,fontWeight=FontWeight.Bold)
                            ShopCategoryTiles(section.categories,3,{onOpen(shopSearchRoute(it.name))},directory=true)
                        }}
                    }
                }
            }
        }
    }
}

@Composable
fun ShopSearchPage(state:LabState,initial:String,onClose:()->Unit,onOpen:(String)->Unit,active:Boolean) {
    var query by rememberSaveable{mutableStateOf(initial)}
    var submitted by rememberSaveable{mutableStateOf(initial)}
    val keyboard=LocalSoftwareKeyboardController.current;val focus=remember{FocusRequester()};val scope=rememberCoroutineScope()
    val list=rememberLazyStaggeredGridState();val page=state.commerce.network?.searchPage
    val products=if(page?.query==submitted.trim())page.items.toList()else emptyList()
    fun search(value:String){query=value;submitted=value.trim();keyboard?.hide();scope.launch{list.scrollToItem(0)}}
    LaunchedEffect(submitted,active){if(active&&submitted.isNotBlank())page?.refresh(submitted,badge=if(submitted==NEW_PRODUCT_BADGE)NEW_PRODUCT_BADGE else "")}
    LaunchedEffect(active){if(active&&submitted.isBlank()){delay(180);focus.requestFocus();keyboard?.show()}}
    CatalogAutoLoad(page,list,products,active&&submitted.isNotBlank())
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).statusBarsPadding().navigationBarsPadding().imePadding()) {
        Row(Modifier.fillMaxWidth().padding(start=4.dp,end=8.dp,top=8.dp,bottom=12.dp),verticalAlignment=Alignment.CenterVertically) {
            IconButton({keyboard?.hide();onClose()}){Icon(Icons.Outlined.ChevronLeft,"返回")}
            RefinedField(query,{query=it.take(80)},Modifier.weight(1f).focusRequester(focus),placeholder={Text("搜索商品、品牌或分类",style=MaterialTheme.typography.bodyMedium)},singleLine=true,keyboardOptions=KeyboardOptions(imeAction=ImeAction.Search),keyboardActions=KeyboardActions(onSearch={search(query)}),trailingIcon=if(query.isNotBlank()){{IconButton({query="";submitted="";focus.requestFocus();keyboard?.show()}){Icon(Icons.Outlined.Cancel,"清除搜索",Modifier.size(18.dp))}}}else null)
            PlainButton({search(query)}){Text("搜索")}
        }
        LazyVerticalStaggeredGrid(columns=StaggeredGridCells.Fixed(2),modifier=Modifier.weight(1f),state=list,contentPadding=PaddingValues(start=6.dp,end=6.dp,top=4.dp,bottom=24.dp),verticalItemSpacing=7.dp,horizontalArrangement=Arrangement.spacedBy(7.dp)) {
            if(submitted.isBlank()) {
                item(span=StaggeredGridItemSpan.FullLine){Column(Modifier.padding(horizontal=12.dp),verticalArrangement=Arrangement.spacedBy(16.dp)){Text("搜索商品",style=MaterialTheme.typography.titleLarge);FlowRow(horizontalArrangement=Arrangement.spacedBy(8.dp),verticalArrangement=Arrangement.spacedBy(8.dp)){(state.commerce.network?.categoryTree.orEmpty().take(8).map{it.name}).distinct().forEach{word->ChoiceChip(false,{search(word)},{Text(word)})}}}}
            } else {
                item(span=StaggeredGridItemSpan.FullLine){Text(submitted,Modifier.padding(horizontal=4.dp),style=MaterialTheme.typography.labelLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                shopProducts(products,{keyboard?.hide();onOpen("product:$it")}){product->val network=state.commerce.network;if(network==null||network.cartBusy)false else network.setCart(product.id,((state.cart[product.id] ?: 0)+1).coerceAtMost(product.limit))}
                catalogStatus(page,products.isEmpty(),"未找到相关商品",scope)
            }
        }
    }
}
