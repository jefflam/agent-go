CREATE TYPE tweet_category AS ENUM (
    'mention',
    'reply',
    'quote',
    'retweet',
    'dm',
    'conversation'
);

CREATE TABLE tweets (
    -- Primary Key
    id TEXT PRIMARY KEY,
    
    -- Core Tweet Fields
    text TEXT NOT NULL,
    conversation_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Author Information
    author_id TEXT NOT NULL,
    author_name TEXT,
    author_username TEXT,
    
    -- Operational Fields (for internal processing)
    category tweet_category NOT NULL,
    processed_at TIMESTAMP NOT NULL,
    last_updated TIMESTAMP NOT NULL,
    process_count INTEGER DEFAULT 0,
    conversation_ref JSONB,
    needs_reply BOOLEAN DEFAULT TRUE,
    is_participating BOOLEAN DEFAULT FALSE,
    
    -- Reply Tracking Fields
    replied_to BOOLEAN DEFAULT FALSE,
    last_reply_id TEXT,
    last_reply_time TIMESTAMP,
    reply_count INTEGER DEFAULT 0,
    unread_replies INTEGER DEFAULT 0,
    in_reply_to_user_id TEXT,
    
    -- Twitter API Response Fields
    attachments JSONB,
    context_annotations JSONB,
    edit_controls JSONB,
    edit_history_tweet_ids TEXT[],
    entities JSONB,
    geo JSONB,
    lang TEXT,
    non_public_metrics JSONB,
    organic_metrics JSONB,
    possibly_sensitive BOOLEAN,
    promoted_metrics JSONB,
    public_metrics JSONB,
    referenced_tweets JSONB,
    reply_settings TEXT,
    source TEXT,
    withheld JSONB
);

-- Core Indexes
CREATE INDEX idx_tweets_conversation_id ON tweets(conversation_id);
CREATE INDEX idx_tweets_author_id ON tweets(author_id);
CREATE INDEX idx_tweets_author_name ON tweets(author_name);
CREATE INDEX idx_tweets_author_username ON tweets(author_username);
CREATE INDEX idx_tweets_category ON tweets(category);
CREATE INDEX idx_tweets_created_at ON tweets(created_at);

-- Reply Processing Indexes
CREATE INDEX idx_tweets_needs_reply ON tweets(needs_reply) WHERE needs_reply = TRUE;
CREATE INDEX idx_tweets_replied_to ON tweets(replied_to);
CREATE INDEX idx_tweets_last_reply_time ON tweets(last_reply_time);