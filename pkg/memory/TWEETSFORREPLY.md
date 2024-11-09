### ✅ Tweets That WILL Be Returned

1. **New Mention Without Reply**
```
User Story: Alice mentions our bot in a new tweet
- Tweet has needs_reply = TRUE
- replied_to = FALSE
- author_id ≠ our_bot_id
Result: Tweet will be returned (Case 1)
```

2. **New Reply in Active Conversation**
```
User Story: Bob replies to a conversation where we previously participated
- is_participating = TRUE
- unread_replies > 0
- author_id ≠ our_bot_id
Result: Tweet will be returned (Case 2)
```

3. **Latest Tweet in Multi-Reply Thread**
```
User Story: Carol and Dave have multiple replies in a conversation we're part of
- conversation_id matches one we're participating in
- created_at is after our last_reply_time
- unread_replies > 0
Result: Only the most recent tweet will be returned (Case 3)
```

### ❌ Tweets That Will NOT Be Returned

1. **Our Own Tweets**
```
User Story: Our bot posts a tweet or reply
- author_id = our_bot_id
Result: Excluded by the author_id check
```

2. **Already Replied Tweets**
```
User Story: A tweet we've already responded to
- replied_to = TRUE
- unread_replies = 0
Result: Excluded because there are no unread replies
```

3. **Non-Participating Conversation**
```
User Story: Users having a conversation without mentioning us
- is_participating = FALSE
- needs_reply = FALSE
Result: Excluded because we're not participating and it doesn't need a reply
```

4. **Older Tweets in Same Conversation**
```
User Story: Multiple tweets in a conversation we're in
- Same conversation_id
- Earlier last_reply_time
Result: Only the most recent tweet is kept, older ones are filtered out
```

5. **Read Replies**

```
User Story: Replies we've already processed
- unread_replies = 0
Result: Filtered out in the final step
```

### Example Timeline Scenario

```

Timeline:
1. @user1: "Hey @bot, how are you?" (RETURNED - new mention)
2. @bot: "I'm good, thanks!"
3. @user1: "That's great!" (RETURNED - new reply in conversation)
4. @user2: "Hello @bot!" (RETURNED - new mention)
5. @user1: "By the way..." (RETURNED - latest in conversation)
6. @user1: "One more thing..." (NOT RETURNED - older tweet in same conversation)
7. @bot: "Thanks everyone!"  (NOT RETURNED - our own tweet)
```

### This functioality ensures we:

- Don't miss new mentions
- Stay engaged in active conversations
- Don't double-reply to the same conversation
- Don't reply to ourselves
- Only focus on the most recent relevant tweets

The key is the combination of the SQL conditions and the post-processing that groups by conversation and filters for unread replies, ensuring we maintain coherent conversations without duplicate responses.