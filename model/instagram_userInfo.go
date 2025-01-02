package model

type UserInfo struct {
	Data struct {
		User struct {
			AIAgentType *string `json:"ai_agent_type"`
			Biography   string  `json:"biography"`
			BioLinks    []struct {
				Title    string `json:"title"`
				LynxURL  string `json:"lynx_url"`
				URL      string `json:"url"`
				LinkType string `json:"link_type"`
			} `json:"bio_links"`
			BiographyWithEntities struct {
				RawText  string `json:"raw_text"`
				Entities []struct {
					Hashtag *struct {
						Name string `json:"name"`
					} `json:"hashtag"`
				} `json:"entities"`
			} `json:"biography_with_entities"`
			BlockedByViewer bool   `json:"blocked_by_viewer"`
			ExternalURL     string `json:"external_url"`
			EdgeFollowedBy  struct {
				Count int `json:"count"`
			} `json:"edge_followed_by"`
			FullName                 string `json:"full_name"`
			IsBusinessAccount        bool   `json:"is_business_account"`
			ProfilePicURL            string `json:"profile_pic_url"`
			ProfilePicURLHD          string `json:"profile_pic_url_hd"`
			Username                 string `json:"username"`
			EdgeOwnerToTimelineMedia struct {
				Count    int `json:"count"`
				PageInfo struct {
					HasNextPage bool   `json:"has_next_page"`
					EndCursor   string `json:"end_cursor"`
				} `json:"page_info"`
				Edges []struct {
					Node struct {
						Typename   string `json:"__typename"`
						ID         string `json:"id"`
						Shortcode  string `json:"shortcode"`
						Dimensions struct {
							Height int `json:"height"`
							Width  int `json:"width"`
						} `json:"dimensions"`
						DisplayURL string `json:"display_url"`
						VideoURL   string `json:"video_url"`
						IsVideo    bool   `json:"is_video"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_owner_to_timeline_media"`
		} `json:"user"`
	} `json:"data"`
}
