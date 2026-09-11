package com.deuterium.app.uilab

/** The caller owns identity/deduplication and keeps this list sorted by [order].
 * Inserting after equal keys preserves append + stable-sort ordering. */
internal fun <T> MutableList<T>.insertInOrder(value:T,order:Comparator<in T>) {
    var low=0;var high=size
    while(low<high){val middle=(low+high).ushr(1);if(order.compare(this[middle],value)<=0)low=middle+1 else high=middle}
    add(low,value)
}

internal fun <T> MutableList<T>.replaceInOrder(index:Int,value:T,order:Comparator<in T>) {
    if(index>=0){
        if(this[index]==value)return
        if(order.compare(this[index],value)==0){this[index]=value;return}
        removeAt(index)
    }
    insertInOrder(value,order)
}
