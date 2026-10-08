package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.lazy.staggeredgrid.*
import androidx.compose.ui.graphics.Brush
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
    LaunchedEffect(Unit){launch{network?.refreshHome()};launch{network?.recommendedPage?.refresh(badge=NEW_PRODUCT_BADGE)}}
    val allProducts=page?.items?.toList().orEmpty()
    val recommendations=network?.recommendedPage?.items?.filter{NEW_PRODUCT_BADGE in it.badges}.orEmpty()
    val featuredIds=remember(recommendations){recommendations.take(12).map{it.id}.toSet()}
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
        LazyVerticalStaggeredGrid(columns=StaggeredGridCells.Fixed(2),modifier=Modifier.weight(1f),state=list,contentPadding=PaddingValues(start=8.dp,end=8.dp,top=4.dp,bottom=115.dp),verticalItemSpacing=8.dp,horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            if(category.isBlank()) {
                val banners=network?.banners?.toList().orEmpty()
                if(banners.isNotEmpty())item(key="banners",span=StaggeredGridItemSpan.FullLine) {
                    LazyRow(horizontalArrangement=Arrangement.spacedBy(8.dp)) {
                        items(banners,key={it.id}){banner->
                            Box(Modifier.fillParentMaxWidth().height(108.dp).clip(RoundedCornerShape(14.dp))) {
                                ServerAssetImage(banner.image,banner.title,Modifier.fillMaxSize(),scale=ContentScale.Crop)
                                Box(Modifier.fillMaxSize().background(Brush.horizontalGradient(listOf(Color.Black.copy(alpha=.5f),Color.Transparent))))
                                Text(banner.title,Modifier.align(Alignment.CenterStart).padding(18.dp).fillMaxWidth(.68f),color=Color.White,fontSize=21.sp,lineHeight=27.sp,fontWeight=FontWeight.Bold,maxLines=2)
                            }
                        }
                    }
                }
                if(network?.homeError!=null)item(span=StaggeredGridItemSpan.FullLine){CatalogMessage("活动暂时无法加载"){scope.launch{network.refreshHome(true)}}}
                if(recommendations.isNotEmpty()) {
                    item(span=StaggeredGridItemSpan.FullLine){ShopSectionTitle(NEW_PRODUCT_BADGE,"查看更多"){onOpen(shopSearchRoute(NEW_PRODUCT_BADGE))}}
                    item(key="recommendations",span=StaggeredGridItemSpan.FullLine){LazyRow(horizontalArrangement=Arrangement.spacedBy(8.dp)){
                        items(recommendations.take(12),key={it.id}){ShopRecommendation(it){onProduct(it.id)}}
                    }}
                }
                if(network?.recommendedPage?.error!=null)item(span=StaggeredGridItemSpan.FullLine){CatalogMessage("新品推荐暂时无法加载"){scope.launch{network.recommendedPage.retry()}}}
            } else if(children.isNotEmpty()) {
                item(span=StaggeredGridItemSpan.FullLine){ShopCategoryTiles(children.take(if(children.size>15)14 else 15),columns=5,onSelect={onOpen(shopSearchRoute(it.name))},more=if(children.size>15){{onOpen("shop-categories:$category")}}else null)}
            }
            if(products.isNotEmpty())item(span=StaggeredGridItemSpan.FullLine){ShopSectionTitle(if(category.isBlank())"逛逛好物"else selectedGroup)}
            shopProducts(products,onProduct){product->
                if(network==null||network.cartBusy)false else network.setCart(product.id,((state.cart[product.id] ?: 0)+1).coerceAtMost(product.limit))
            }
            catalogStatus(page,products.isEmpty()&&(category.isNotBlank()||recommendations.isEmpty()),"暂无上架商品",scope)
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

@Composable
private fun ShopSectionTitle(title:String,action:String?=null,onAction:()->Unit={}) {
    Row(Modifier.fillMaxWidth().heightIn(min=36.dp).padding(start=4.dp,end=2.dp),verticalAlignment=Alignment.CenterVertically) {
        Text(title,Modifier.weight(1f),fontSize=19.sp,fontWeight=FontWeight.Bold)
        if(action!=null)Row(Modifier.clickable(onClick=onAction).padding(vertical=10.dp,horizontal=6.dp),verticalAlignment=Alignment.CenterVertically){Text(action,fontSize=12.sp,color=MaterialTheme.colorScheme.onSurfaceVariant);Icon(Icons.Outlined.ChevronRight,null,Modifier.size(16.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}
    }
}

@Composable
private fun ShopRecommendation(product:ShopProduct,onClick:()->Unit) {
    Surface(onClick=onClick,modifier=Modifier.width(164.dp),shape=RoundedCornerShape(12.dp),color=MaterialTheme.colorScheme.surface) {
        Column {
            ShopSubjectImage(product.photos.firstOrNull(),product.name,Modifier.fillMaxWidth().height(112.dp))
            Column(Modifier.padding(horizontal=10.dp,vertical=8.dp),verticalArrangement=Arrangement.spacedBy(4.dp)) {
                Text(product.name,fontSize=14.sp,fontWeight=FontWeight.Medium,maxLines=1,overflow=TextOverflow.Ellipsis)
                Row(verticalAlignment=Alignment.Bottom){AdaptiveMoney(credit(product.price),Modifier.weight(1f),color=MaterialTheme.colorScheme.primary,size=19.sp);Text("信用点",fontSize=10.sp,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            }
        }
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
    LaunchedEffect(added){if(added){delay(1000);added=false}}
    Surface(onClick=onClick,modifier=Modifier.fillMaxWidth(),shape=RoundedCornerShape(12.dp),color=MaterialTheme.colorScheme.surface) {
        Column {
            Box {
                ShopSubjectImage(product.photos.firstOrNull(),product.name,Modifier.fillMaxWidth().aspectRatio(.88f))
                if(NEW_PRODUCT_BADGE in product.badges)Text("新品",Modifier.padding(8.dp).clip(RoundedCornerShape(5.dp)).background(MaterialTheme.colorScheme.primaryContainer).padding(horizontal=6.dp,vertical=3.dp),fontSize=10.sp,fontWeight=FontWeight.SemiBold,color=MaterialTheme.colorScheme.primary)
            }
            Column(Modifier.padding(start=10.dp,end=8.dp,top=9.dp,bottom=8.dp),verticalArrangement=Arrangement.spacedBy(5.dp)) {
                Text(product.name,fontSize=15.sp,lineHeight=21.sp,fontWeight=FontWeight.Medium,maxLines=2,overflow=TextOverflow.Ellipsis)
                Text(product.category,fontSize=11.sp,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
                Row(verticalAlignment=Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        AdaptiveMoney(credit(product.price),Modifier.fillMaxWidth(),color=MaterialTheme.colorScheme.primary,size=22.sp)
                        Text("信用点",fontSize=10.sp,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    IconButton({if(!adding)scope.launch{adding=true;try{added=onAdd()}finally{adding=false}}},Modifier.size(44.dp),enabled=!adding&&product.stock>0){
                        Box(Modifier.size(30.dp).background(MaterialTheme.colorScheme.primaryContainer,CircleShape),contentAlignment=Alignment.Center){Icon(if(added)Icons.Outlined.Check else Icons.Outlined.Add,"加入购物袋：${product.name}",Modifier.size(21.dp),tint=MaterialTheme.colorScheme.primary)}
                    }
                }
                if(product.originalPrice>product.price)Text(credit(product.originalPrice),fontSize=11.sp,textDecoration=TextDecoration.LineThrough,color=MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
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
private fun ShopCategoryTiles(categories:List<ShopCategory>,columns:Int,onSelect:(ShopCategory)->Unit,more:(()->Unit)?=null) {
    val cells:List<ShopCategory?> = categories+if(more!=null)listOf(null)else emptyList()
    Column(Modifier.padding(horizontal=10.dp,vertical=8.dp),verticalArrangement=Arrangement.spacedBy(12.dp)) {
        cells.chunked(columns).forEach{row->Row(horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            row.forEach{c->Column(Modifier.weight(1f).clip(RoundedCornerShape(16.dp)).clickable{if(c==null)more?.invoke()else onSelect(c)}.padding(vertical=6.dp),horizontalAlignment=Alignment.CenterHorizontally) {
                val image=if(c==null)null else c.image ?: ShopCatalog.firstOrNull{it.categoryId==c.id}?.photos?.firstOrNull()
                if(image!=null)ShopSubjectImage(image,c?.name.orEmpty(),Modifier.fillMaxWidth().aspectRatio(1f).clip(RoundedCornerShape(12.dp)))
                else Box(Modifier.fillMaxWidth().aspectRatio(1f).background(MaterialTheme.colorScheme.surface,CircleShape),contentAlignment=Alignment.Center){Icon(if(c==null)Icons.Outlined.MoreHoriz else Icons.Outlined.Widgets,null,Modifier.size(25.dp),tint=MaterialTheme.colorScheme.primary)}
                Text(c?.name ?: "更多",Modifier.padding(top=7.dp),fontSize=12.sp,lineHeight=17.sp,maxLines=2,minLines=2,overflow=TextOverflow.Ellipsis,textAlign=TextAlign.Center)
            }}
            repeat(columns-row.size){Spacer(Modifier.weight(1f))}
        }}
    }
}

@Composable
fun ShopCategoriesPage(state:LabState,initial:String,onClose:()->Unit,onOpen:(String)->Unit) {
    val network=state.commerce.network
    LaunchedEffect(Unit){network?.ensureMetadata()}
    val categories=network?.categoryTree?.toList().orEmpty();val roots=remember(categories){shopRootCategories(categories)}
    var selected by rememberSaveable {mutableStateOf(initial)}
    val current=roots.firstOrNull{it.id==selected} ?: roots.firstOrNull()
    val children=shopGroupChildren(categories,current?.name.orEmpty())
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).statusBarsPadding().navigationBarsPadding()) {
        Row(Modifier.fillMaxWidth().height(56.dp),verticalAlignment=Alignment.CenterVertically){IconButton(onClose){Icon(Icons.Outlined.ChevronLeft,"返回")};Text("分类",Modifier.weight(1f),textAlign=TextAlign.Center,style=MaterialTheme.typography.titleLarge);IconButton({onOpen("search:Shop")}){Icon(Icons.Outlined.Search,"搜索商品")}}
        if(roots.isEmpty()){val scope=rememberCoroutineScope();CatalogMessage(network?.shopError ?: "暂无商品分类"){scope.launch{network?.refreshStore(force=true)}}}
        Row(Modifier.weight(1f)) {
            LazyColumn(Modifier.width(88.dp).fillMaxHeight(),contentPadding=PaddingValues(vertical=8.dp)) {
                items(roots,key={it.id}){c->Surface(onClick={selected=c.id},color=if(current?.id==c.id)MaterialTheme.colorScheme.surface else MaterialTheme.colorScheme.background){Text(c.name,Modifier.fillMaxWidth().padding(horizontal=10.dp,vertical=22.dp),color=if(current?.id==c.id)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.titleSmall,textAlign=TextAlign.Center)}}
            }
            key(current?.id){LazyColumn(Modifier.weight(1f).fillMaxHeight().background(MaterialTheme.colorScheme.surface),contentPadding=PaddingValues(top=8.dp,bottom=24.dp),verticalArrangement=Arrangement.spacedBy(4.dp)) {
                current?.let{c->item{Row(Modifier.fillMaxWidth().clickable{onOpen(shopSearchRoute(c.name))}.padding(horizontal=14.dp,vertical=12.dp),verticalAlignment=Alignment.CenterVertically){Text(c.name,Modifier.weight(1f),style=MaterialTheme.typography.titleLarge);Icon(Icons.Outlined.ChevronRight,"查看全部商品")}}}
                items((if(children.isEmpty())listOfNotNull(current)else children).chunked(12)){group->ShopCategoryTiles(group,3,{onOpen(shopSearchRoute(it.name))})}
            }}
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
        LazyVerticalStaggeredGrid(columns=StaggeredGridCells.Fixed(2),modifier=Modifier.weight(1f),state=list,contentPadding=PaddingValues(start=8.dp,end=8.dp,top=4.dp,bottom=24.dp),verticalItemSpacing=8.dp,horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            if(submitted.isBlank()) {
                item(span=StaggeredGridItemSpan.FullLine){Column(Modifier.padding(horizontal=12.dp),verticalArrangement=Arrangement.spacedBy(16.dp)){Text("搜索商品",style=MaterialTheme.typography.titleLarge);FlowRow(horizontalArrangement=Arrangement.spacedBy(8.dp),verticalArrangement=Arrangement.spacedBy(8.dp)){(listOf(NEW_PRODUCT_BADGE)+state.commerce.network?.categoryTree.orEmpty().take(8).map{it.name}).distinct().forEach{word->ChoiceChip(false,{search(word)},{Text(word)})}}}}
            } else {
                item(span=StaggeredGridItemSpan.FullLine){Text(submitted,Modifier.padding(horizontal=4.dp),style=MaterialTheme.typography.labelLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                shopProducts(products,{keyboard?.hide();onOpen("product:$it")}){product->val network=state.commerce.network;if(network==null||network.cartBusy)false else network.setCart(product.id,((state.cart[product.id] ?: 0)+1).coerceAtMost(product.limit))}
                catalogStatus(page,products.isEmpty(),"未找到相关商品",scope)
            }
        }
    }
}
