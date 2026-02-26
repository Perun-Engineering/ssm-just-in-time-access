package slack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/slack-go/slack"
)

// CacheEntry represents a cached group membership entry
type CacheEntry struct {
	Members    []string
	Expiration time.Time
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.Expiration)
}

// GroupMembershipCache caches Slack user group memberships
type GroupMembershipCache struct {
	client *slack.Client
	cache  map[string]*CacheEntry
	mutex  sync.RWMutex
	ttl    time.Duration
}

// NewGroupMembershipCache creates a new group membership cache
func NewGroupMembershipCache(botToken string, ttl time.Duration) *GroupMembershipCache {
	return &GroupMembershipCache{
		client: slack.New(botToken),
		cache:  make(map[string]*CacheEntry),
		ttl:    ttl,
	}
}

// GetMembers retrieves members of a user group, using cache if available
func (c *GroupMembershipCache) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	// Check cache first
	c.mutex.RLock()
	entry, exists := c.cache[groupID]
	c.mutex.RUnlock()

	if exists && !entry.IsExpired() {
		return entry.Members, nil
	}

	// Cache miss or expired, fetch from Slack API
	members, err := c.fetchMembersFromSlack(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group members: %w", err)
	}

	// Update cache
	c.mutex.Lock()
	c.cache[groupID] = &CacheEntry{
		Members:    members,
		Expiration: time.Now().Add(c.ttl),
	}
	c.mutex.Unlock()

	return members, nil
}

// IsMember checks if a user is a member of a user group
func (c *GroupMembershipCache) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	members, err := c.GetMembers(ctx, groupID)
	if err != nil {
		return false, err
	}

	for _, member := range members {
		if member == userID {
			return true, nil
		}
	}

	return false, nil
}

// Invalidate removes a group from the cache
func (c *GroupMembershipCache) Invalidate(groupID string) {
	c.mutex.Lock()
	delete(c.cache, groupID)
	c.mutex.Unlock()
}

// InvalidateAll clears the entire cache
func (c *GroupMembershipCache) InvalidateAll() {
	c.mutex.Lock()
	c.cache = make(map[string]*CacheEntry)
	c.mutex.Unlock()
}

// fetchMembersFromSlack retrieves group members from Slack API
func (c *GroupMembershipCache) fetchMembersFromSlack(ctx context.Context, groupID string) ([]string, error) {
	userGroup, err := c.client.GetUserGroupMembersContext(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user group members from Slack: %w", err)
	}

	return userGroup, nil
}
