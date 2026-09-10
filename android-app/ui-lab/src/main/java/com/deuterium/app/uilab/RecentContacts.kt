package com.deuterium.app.uilab

data class ContactPreview(val person:PlayerProfile,val latest:ChatLine?,val followed:Boolean)

/** Opening an empty conversation or typing a draft does not make somebody a recent contact. */
fun recentContacts(people:List<PlayerProfile>,followed:Set<String>,conversations:Map<String,List<ChatLine>>,searchMode:Boolean=false):List<ContactPreview> =
    people.distinctBy{it.playerRef.ifBlank{it.name}}.mapNotNull{person->
        val latest=conversations[person.name]?.asSequence()?.filter{it.remoteId.isNotBlank()}?.maxByOrNull{it.serverAt}
        val favorite=person.name in followed
        if(!searchMode&&latest==null&&!favorite)null else ContactPreview(person,latest,favorite)
    }.sortedWith(compareByDescending<ContactPreview>{it.followed}.thenByDescending{it.latest?.serverAt ?: Long.MIN_VALUE}.thenBy{it.person.name.lowercase()})
