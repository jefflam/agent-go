package twitter

import "fmt"

// Tweet represents a Twitter post with all v2 API fields
type Tweet struct {
	// Required fields
	ID   string `json:"id"`
	Text string `json:"text"`

	// Optional fields
	Attachments struct {
		MediaKeys []string `json:"media_keys,omitempty"`
		PollIDs   []string `json:"poll_ids,omitempty"`
	} `json:"attachments,omitempty"`
	AuthorID           string `json:"author_id,omitempty"`
	ContextAnnotations []struct {
		Domain struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
		} `json:"domain"`
		Entity struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
		} `json:"entity"`
	} `json:"context_annotations,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	EditControls   struct {
		EditableUntil  string `json:"editable_until,omitempty"`
		EditsRemaining int    `json:"edits_remaining,omitempty"`
		IsEditEligible bool   `json:"is_edit_eligible,omitempty"`
	} `json:"edit_controls,omitempty"`
	EditHistoryTweetIDs []string `json:"edit_history_tweet_ids,omitempty"`
	Entities            struct {
		Annotations []struct {
			Start          int     `json:"start"`
			End            int     `json:"end"`
			Probability    float64 `json:"probability"`
			Type           string  `json:"type"`
			NormalizedText string  `json:"normalized_text"`
		} `json:"annotations,omitempty"`
		Cashtags []struct {
			Start int    `json:"start"`
			End   int    `json:"end"`
			Tag   string `json:"tag"`
		} `json:"cashtags,omitempty"`
		Hashtags []struct {
			Start int    `json:"start"`
			End   int    `json:"end"`
			Tag   string `json:"tag"`
		} `json:"hashtags,omitempty"`
		Mentions []struct {
			Start    int    `json:"start"`
			End      int    `json:"end"`
			Username string `json:"username"`
			ID       string `json:"id"`
		} `json:"mentions,omitempty"`
		URLs []struct {
			Start       int    `json:"start"`
			End         int    `json:"end"`
			URL         string `json:"url"`
			ExpandedURL string `json:"expanded_url"`
			DisplayURL  string `json:"display_url"`
			Status      string `json:"status,omitempty"`
			Title       string `json:"title,omitempty"`
			Description string `json:"description,omitempty"`
			UnwoundURL  string `json:"unwound_url,omitempty"`
		} `json:"urls,omitempty"`
	} `json:"entities,omitempty"`
	Geo struct {
		Coordinates struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"coordinates,omitempty"`
		PlaceID string `json:"place_id,omitempty"`
	} `json:"geo,omitempty"`
	InReplyToUserID  string `json:"in_reply_to_user_id,omitempty"`
	Lang             string `json:"lang,omitempty"`
	NonPublicMetrics struct {
		ImpressionCount   int `json:"impression_count"`
		URLLinkClicks     int `json:"url_link_clicks"`
		UserProfileClicks int `json:"user_profile_clicks"`
	} `json:"non_public_metrics,omitempty"`
	OrganicMetrics struct {
		ImpressionCount   int `json:"impression_count"`
		LikeCount         int `json:"like_count"`
		ReplyCount        int `json:"reply_count"`
		RetweetCount      int `json:"retweet_count"`
		URLLinkClicks     int `json:"url_link_clicks"`
		UserProfileClicks int `json:"user_profile_clicks"`
	} `json:"organic_metrics,omitempty"`
	PossiblySensitive bool `json:"possibly_sensitive,omitempty"`
	PromotedMetrics   struct {
		ImpressionCount   int `json:"impression_count"`
		LikeCount         int `json:"like_count"`
		ReplyCount        int `json:"reply_count"`
		RetweetCount      int `json:"retweet_count"`
		URLLinkClicks     int `json:"url_link_clicks"`
		UserProfileClicks int `json:"user_profile_clicks"`
	} `json:"promoted_metrics,omitempty"`
	PublicMetrics struct {
		RetweetCount int `json:"retweet_count"`
		ReplyCount   int `json:"reply_count"`
		LikeCount    int `json:"like_count"`
		QuoteCount   int `json:"quote_count"`
	} `json:"public_metrics,omitempty"`
	ReferencedTweets []struct {
		Type string `json:"type"` // "retweeted" or "quoted" or "replied_to"
		ID   string `json:"id"`
	} `json:"referenced_tweets,omitempty"`
	ReplySettings string `json:"reply_settings,omitempty"` // "everyone", "mentionedUsers", "following"
	Source        string `json:"source,omitempty"`
	Withheld      struct {
		Copyright    bool     `json:"copyright,omitempty"`
		CountryCodes []string `json:"country_codes,omitempty"`
		Scope        string   `json:"scope,omitempty"` // "tweet" or "user"
	} `json:"withheld,omitempty"`
}

// TweetResponse represents the Twitter API response format
type TweetResponse struct {
	Data     []Tweet        `json:"data"`
	Includes *TweetIncludes `json:"includes,omitempty"`
	Errors   []TwitterError `json:"errors,omitempty"`
	Meta     *Meta          `json:"meta,omitempty"`
}

// TweetIncludes contains the expanded objects in the response
type TweetIncludes struct {
	Users  []User  `json:"users,omitempty"`
	Tweets []Tweet `json:"tweets,omitempty"`
	Media  []Media `json:"media,omitempty"`
	Places []Place `json:"places,omitempty"`
	Polls  []Poll  `json:"polls,omitempty"`
}

// TwitterError represents an error returned by the Twitter API
type TwitterError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *TwitterError) Error() string {
	return fmt.Sprintf("Twitter API error %d: %s", e.Code, e.Message)
}

// User represents a Twitter user object
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	CreatedAt   string `json:"created_at,omitempty"`
	Description string `json:"description,omitempty"`
	Entities    struct {
		URL struct {
			URLs []struct {
				Start       int    `json:"start"`
				End         int    `json:"end"`
				URL         string `json:"url"`
				ExpandedURL string `json:"expanded_url"`
				DisplayURL  string `json:"display_url"`
			} `json:"urls,omitempty"`
		} `json:"url,omitempty"`
		Description struct {
			URLs []struct {
				Start       int    `json:"start"`
				End         int    `json:"end"`
				URL         string `json:"url"`
				ExpandedURL string `json:"expanded_url"`
				DisplayURL  string `json:"display_url"`
			} `json:"urls,omitempty"`
		} `json:"description,omitempty"`
	} `json:"entities,omitempty"`
	Location        string `json:"location,omitempty"`
	PinnedTweetID   string `json:"pinned_tweet_id,omitempty"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
	Protected       bool   `json:"protected,omitempty"`
	PublicMetrics   struct {
		FollowersCount int `json:"followers_count"`
		FollowingCount int `json:"following_count"`
		TweetCount     int `json:"tweet_count"`
		ListedCount    int `json:"listed_count"`
	} `json:"public_metrics,omitempty"`
	URL      string `json:"url,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Withheld struct {
		CountryCodes []string `json:"country_codes,omitempty"`
		Scope        string   `json:"scope,omitempty"`
	} `json:"withheld,omitempty"`
}

// Media represents a media object attached to a Tweet
type Media struct {
	MediaKey        string `json:"media_key"`
	Type            string `json:"type"` // "animated_gif", "photo", "video"
	URL             string `json:"url,omitempty"`
	DurationMS      int    `json:"duration_ms,omitempty"`
	Height          int    `json:"height,omitempty"`
	Width           int    `json:"width,omitempty"`
	PreviewImageURL string `json:"preview_image_url,omitempty"`
	PublicMetrics   struct {
		ViewCount int `json:"view_count,omitempty"`
	} `json:"public_metrics,omitempty"`
	AltText  string `json:"alt_text,omitempty"`
	Variants []struct {
		BitRate     int    `json:"bit_rate,omitempty"`
		ContentType string `json:"content_type"`
		URL         string `json:"url"`
	} `json:"variants,omitempty"`
}

// Place represents a location tagged in a Tweet
type Place struct {
	ID          string `json:"id"`
	FullName    string `json:"full_name"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	Geo         struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
		Properties  struct{}    `json:"properties"`
	} `json:"geo"`
	Name      string `json:"name"`
	PlaceType string `json:"place_type"`
}

// Poll represents a poll attached to a Tweet
type Poll struct {
	ID      string `json:"id"`
	Options []struct {
		Position int    `json:"position"`
		Label    string `json:"label"`
		Votes    int    `json:"votes"`
	} `json:"options"`
	DurationMinutes int    `json:"duration_minutes"`
	EndDateTime     string `json:"end_datetime"`
	VotingStatus    string `json:"voting_status"`
}

// Meta contains information about the response
type Meta struct {
	ResultCount     int    `json:"result_count,omitempty"`
	NextToken       string `json:"next_token,omitempty"`
	PreviousToken   string `json:"previous_token,omitempty"`
	NewestID        string `json:"newest_id,omitempty"`
	OldestID        string `json:"oldest_id,omitempty"`
	TotalTweetCount int    `json:"total_tweet_count,omitempty"`
	Sent            string `json:"sent,omitempty"`
	Summary         struct {
		Created    int `json:"created"`
		NotCreated int `json:"not_created"`
		Valid      int `json:"valid"`
		Invalid    int `json:"invalid"`
	} `json:"summary,omitempty"`
}

// List represents a Twitter List object
type List struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	FollowerCount int    `json:"follower_count"`
	MemberCount   int    `json:"member_count"`
	Private       bool   `json:"private"`
	OwnerID       string `json:"owner_id"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// Space represents a Twitter Space
type Space struct {
	ID               string   `json:"id"`
	State            string   `json:"state"`
	CreatedAt        string   `json:"created_at"`
	EndedAt          string   `json:"ended_at,omitempty"`
	HostIDs          []string `json:"host_ids"`
	Lang             string   `json:"lang"`
	IsTicketed       bool     `json:"is_ticketed"`
	InvitedUserIDs   []string `json:"invited_user_ids,omitempty"`
	ParticipantCount int      `json:"participant_count"`
	ScheduledStart   string   `json:"scheduled_start,omitempty"`
	SpeakerIDs       []string `json:"speaker_ids"`
	StartedAt        string   `json:"started_at,omitempty"`
	Title            string   `json:"title"`
	TopicIDs         []string `json:"topic_ids,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
}

// Topic represents a Twitter topic
type Topic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// DirectMessage represents a Twitter DM
type DirectMessage struct {
	ID             string `json:"id"`
	Text           string `json:"text"`
	EventType      string `json:"event_type"`
	CreatedAt      string `json:"created_at"`
	SenderID       string `json:"sender_id"`
	RecipientID    string `json:"recipient_id"`
	ConversationID string `json:"conversation_id"`
	Attachments    []struct {
		MediaType string `json:"media_type"`
		MediaID   string `json:"media_id"`
	} `json:"attachments,omitempty"`
}

// Trend represents a Twitter trending topic
type Trend struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	TweetVolume int    `json:"tweet_volume,omitempty"`
	PlaceType   string `json:"place_type,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Woeid       int    `json:"woeid"`
}

// Compliance represents a Twitter compliance event
type Compliance struct {
	ID          string `json:"id"`
	EventType   string `json:"event_type"`
	CreatedAt   string `json:"created_at"`
	TweetID     string `json:"tweet_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	UploadID    string `json:"upload_id,omitempty"`
	ResumeToken string `json:"resume_token,omitempty"`
}

// RulesLookupResponse represents the response for rules lookup
type RulesLookupResponse struct {
	Data  []Rule        `json:"data,omitempty"`
	Meta  *Meta         `json:"meta,omitempty"`
	Error *TwitterError `json:"error,omitempty"`
}

// Rule represents a filtering rule for filtered streams
type Rule struct {
	ID    string `json:"id"`
	Value string `json:"value"`
	Tag   string `json:"tag,omitempty"`
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	Data          *Tweet         `json:"data"`
	Includes      *TweetIncludes `json:"includes,omitempty"`
	MatchingRules []struct {
		ID  string `json:"id"`
		Tag string `json:"tag,omitempty"`
	} `json:"matching_rules,omitempty"`
}

// TweetCountsResponse represents the response from tweet counts endpoint
type TweetCountsResponse struct {
	Data []struct {
		Start      string `json:"start"`
		End        string `json:"end"`
		TweetCount int    `json:"tweet_count"`
	} `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// ListResponse represents the response for list operations
type ListResponse struct {
	Data  *List         `json:"data"`
	Meta  *Meta         `json:"meta,omitempty"`
	Error *TwitterError `json:"error,omitempty"`
}

// ListMembersResponse represents the response for list members
type ListMembersResponse struct {
	Data     []User         `json:"data"`
	Includes *TweetIncludes `json:"includes,omitempty"`
	Meta     *Meta          `json:"meta,omitempty"`
	Error    *TwitterError  `json:"error,omitempty"`
}

// ConversationResponse represents the response from the conversation lookup endpoint
type ConversationResponse struct {
	Data     []Tweet        `json:"data"`
	Includes *TweetIncludes `json:"includes,omitempty"`
	Errors   []TwitterError `json:"errors,omitempty"`
	Meta     *Meta          `json:"meta,omitempty"`
}
