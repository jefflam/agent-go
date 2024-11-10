# Twitter Agent User Stories (Current Implementation)

## 1. Initial Mentions
```gherkin
As a Twitter Agent
When I receive a new mention (@CatLordLaffy)
Then I should:
  - Save the tweet with needs_reply = true
  - Set category = 'mention'
  - Set is_participating = false
  - Store author details and metadata
  - Queue it for response
```

## 2. Conversation Replies
```gherkin
As a Twitter Agent
When someone replies to a conversation I'm participating in
Then I should:
  - Increment unread_replies on the parent tweet
  - Set needs_reply = true on the parent
  - Update last_updated timestamp
  - Set is_participating = true for all conversation tweets
  - Store the reply with proper context
```

## 3. Agent Response Tracking
```gherkin
As a Twitter Agent
When I reply to a tweet
Then I should:
  - Store my reply with:
    * category = 'reply'
    * is_participating = true
    * needs_reply = false
    * replied_to = false
  - Update the original tweet:
    * Set replied_to = true
    * Set needs_reply = false
    * Update last_reply_id and time
```

## 4. Conversation Context
```gherkin
As a Twitter Agent
When I process a conversation
Then I should:
  - Load all tweets in the conversation_id
  - Order them chronologically
  - Track participation status
  - Maintain reply chains
  - Include referenced tweets
```

## 5. Reply Detection
```gherkin
As a Twitter Agent
When checking for tweets needing replies
Then I should identify tweets where:
  - New mentions haven't been replied to
  - Conversations have unread replies
  - New activity exists after my last reply
  - Referenced tweets need context
```

## 6. Conversation State
```gherkin
As a Twitter Agent
When tracking conversation state
Then I should:
  - Update is_participating flags
  - Track last_reply_time
  - Maintain unread_replies count
  - Track conversation_id relationships
```

## 7. Database Operations
```gherkin
As a Twitter Agent
When performing database operations
Then I should:
  - Use proper locking (mu.Lock/RLock)
  - Execute within transactions
  - Log operations with proper context
  - Handle errors appropriately
```

## 8. Tweet Categories
```gherkin
As a Twitter Agent
When categorizing tweets
Then I should properly identify:
  - Mentions (new @mentions)
  - Replies (responses in threads)
  - Conversations (threaded discussions)
  - Quotes and Retweets
```

## 9. Conversation Participation
```gherkin
As a Twitter Agent
When joining a conversation
Then I should:
  - Set is_participating = true
  - Update all related tweets
  - Track conversation context
  - Maintain reply chains
```

## 10. Tweet Recall
```gherkin
As a Twitter Agent
When recalling tweets for processing
Then I should:
  - Exclude my own tweets (author_id != userID)
  - Include unreplied mentions
  - Include active conversations
  - Include proper context
  - Order by creation time
```

These stories reflect the current implementation's capabilities without suggesting modifications. Would you like me to elaborate on any particular story or aspect of the current implementation?