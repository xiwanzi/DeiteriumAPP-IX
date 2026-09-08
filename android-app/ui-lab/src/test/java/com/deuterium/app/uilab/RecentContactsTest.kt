package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test

class RecentContactsTest {
    private fun person(name:String)=PlayerProfile(name,playerRef="player_$name")
    private fun message(at:Long,mine:Boolean=false,confirmed:Boolean=true)=ChatLine(at,"sender","message",mine,time="23:59",remoteId=if(confirmed)"message_$at" else "",serverAt=at)

    @Test fun onlySentOrReceivedPrivateMessagesCreateRegularContacts(){
        val people=listOf("Never","Empty","Draft","Sent","Received").map(::person)
        val messages=mapOf("Empty" to emptyList(),"Draft" to listOf(message(100,confirmed=false)),"Sent" to listOf(message(200,mine=true)),"Received" to listOf(message(300)))
        assertEquals(listOf("Received","Sent"),recentContacts(people,emptySet(),messages).map{it.person.name})
    }
    @Test fun favoritesStayAboveMoreRecentOrdinaryContacts(){
        val people=listOf("Favorite","FavoriteNew","Recent","Old").map(::person)
        val messages=mapOf("FavoriteNew" to listOf(message(20)),"Recent" to listOf(message(300)),"Old" to listOf(message(10)))
        assertEquals(listOf("FavoriteNew","Favorite","Recent","Old"),recentContacts(people,setOf("Favorite","FavoriteNew"),messages).map{it.person.name})
    }
    @Test fun incomingAndOutgoingActivityReorderByTimestampAcrossDays(){
        val people=listOf("Alice","Bob").map(::person)
        val messages=mutableMapOf("Alice" to listOf(message(86_399)),"Bob" to listOf(message(86_401).copy(time="00:00")))
        assertEquals("Bob",recentContacts(people,emptySet(),messages).first().person.name)
        messages["Alice"]=listOf(message(90_000,mine=true),message(1))
        assertEquals("Alice",recentContacts(people,emptySet(),messages).first().person.name)
        assertEquals(90_000,recentContacts(people,emptySet(),messages).first().latest!!.serverAt)
    }
    @Test fun searchModeIncludesTheEntireMatchingDirectory(){
        val people=listOf("Never","Sent","Favorite").map(::person)
        val results=recentContacts(people,setOf("Favorite"),mapOf("Sent" to listOf(message(1))),searchMode=true)
        assertEquals(listOf("Favorite","Sent","Never"),results.map{it.person.name})
    }
    @Test fun stableIdentityIsNotDuplicated(){
        val person=person("Alice")
        assertEquals(1,recentContacts(listOf(person,person),setOf("Alice"),emptyMap()).size)
    }
}
