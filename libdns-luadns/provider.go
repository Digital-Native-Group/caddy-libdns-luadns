// Package luadns implements a DNS record management client compatible
// with the libdns interfaces for Lua DNS (https://www.luadns.com/).
package luadns

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Lua DNS.
type Provider struct {
	// Email is the email address associated with your Lua DNS account
	Email string `json:"email,omitempty"`

	// APIKey is your Lua DNS API key from https://www.luadns.com/api_keys
	APIKey string `json:"api_key,omitempty"`

	// client is the internal HTTP client for API communication
	client *Client

	// zoneCache maps zone names to zone IDs to minimize API calls
	zoneCache   map[string]int
	zoneCacheMu sync.RWMutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	apiRecords, err := p.client.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	records := make([]libdns.Record, 0, len(apiRecords))
	for _, r := range apiRecords {
		records = append(records, toLibdnsRecord(r, zone))
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	var created []libdns.Record
	for _, rec := range records {
		apiRec := fromLibdnsRecord(rec, zone)

		createdRec, err := p.client.CreateRecord(ctx, zoneID, apiRec)
		if err != nil {
			return created, fmt.Errorf("failed to create record %s: %w", rec.Name, err)
		}

		created = append(created, toLibdnsRecord(createdRec, zone))
	}

	return created, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	// Get existing records to find matches
	existingRecords, err := p.client.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	var updated []libdns.Record
	for _, rec := range records {
		// Try to find existing record with same name and type
		var existingID int
		for _, existing := range existingRecords {
			if matchesRecord(existing, rec, zone) {
				existingID = existing.ID
				break
			}
		}

		apiRec := fromLibdnsRecord(rec, zone)

		if existingID > 0 {
			// Update existing record
			updatedRec, err := p.client.UpdateRecord(ctx, zoneID, existingID, apiRec)
			if err != nil {
				return updated, fmt.Errorf("failed to update record %s: %w", rec.Name, err)
			}
			updated = append(updated, toLibdnsRecord(updatedRec, zone))
		} else {
			// Create new record
			createdRec, err := p.client.CreateRecord(ctx, zoneID, apiRec)
			if err != nil {
				return updated, fmt.Errorf("failed to create record %s: %w", rec.Name, err)
			}
			updated = append(updated, toLibdnsRecord(createdRec, zone))
		}
	}

	return updated, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}

	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	// Get existing records to find IDs
	existingRecords, err := p.client.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	var deleted []libdns.Record
	for _, rec := range records {
		// Find the record ID
		var recordID int
		for _, existing := range existingRecords {
			if matchesRecord(existing, rec, zone) {
				recordID = existing.ID
				break
			}
		}

		if recordID == 0 {
			// Record not found, skip
			continue
		}

		err := p.client.DeleteRecord(ctx, zoneID, recordID)
		if err != nil {
			return deleted, fmt.Errorf("failed to delete record %s: %w", rec.Name, err)
		}

		deleted = append(deleted, rec)
	}

	return deleted, nil
}

// ensureClient creates the HTTP client if it doesn't exist
func (p *Provider) ensureClient() error {
	if p.Email == "" {
		return fmt.Errorf("email is required")
	}
	if p.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if p.client == nil {
		p.client = NewClient(p.Email, p.APIKey)
	}

	return nil
}

// getZoneID retrieves the zone ID for a given zone name, using cache when possible
func (p *Provider) getZoneID(ctx context.Context, zone string) (int, error) {
	zone = strings.TrimSuffix(zone, ".")

	// Check cache first
	p.zoneCacheMu.RLock()
	if p.zoneCache != nil {
		if id, ok := p.zoneCache[zone]; ok {
			p.zoneCacheMu.RUnlock()
			return id, nil
		}
	}
	p.zoneCacheMu.RUnlock()

	// Fetch from API
	zones, err := p.client.ListZones(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list zones: %w", err)
	}

	// Find the zone
	var zoneID int
	for _, z := range zones {
		if strings.TrimSuffix(z.Name, ".") == zone {
			zoneID = z.ID
			break
		}
	}

	if zoneID == 0 {
		return 0, fmt.Errorf("zone %s not found", zone)
	}

	// Update cache
	p.zoneCacheMu.Lock()
	if p.zoneCache == nil {
		p.zoneCache = make(map[string]int)
	}
	p.zoneCache[zone] = zoneID
	p.zoneCacheMu.Unlock()

	return zoneID, nil
}

// toLibdnsRecord converts a Lua DNS API record to a libdns record
func toLibdnsRecord(r Record, zone string) libdns.Record {
	name := strings.TrimSuffix(r.Name, "."+zone)
	name = strings.TrimSuffix(name, ".")

	return libdns.Record{
		ID:    fmt.Sprintf("%d", r.ID),
		Type:  r.Type,
		Name:  name,
		Value: r.Content,
		TTL:   time.Duration(r.TTL) * time.Second,
	}
}

// fromLibdnsRecord converts a libdns record to a Lua DNS API record
func fromLibdnsRecord(r libdns.Record, zone string) Record {
	// Construct FQDN
	name := r.Name
	if name == "@" || name == "" {
		name = zone
	} else if !strings.HasSuffix(name, ".") {
		name = name + "." + zone
	}
	name = strings.TrimSuffix(name, ".")

	ttl := int(r.TTL.Seconds())
	if ttl == 0 {
		ttl = 3600 // Default TTL
	}

	return Record{
		Name:    name,
		Type:    r.Type,
		Content: r.Value,
		TTL:     ttl,
	}
}

// matchesRecord checks if an API record matches a libdns record
func matchesRecord(apiRec Record, libRec libdns.Record, zone string) bool {
	libAsAPI := fromLibdnsRecord(libRec, zone)

	return apiRec.Name == libAsAPI.Name &&
		apiRec.Type == libAsAPI.Type
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
