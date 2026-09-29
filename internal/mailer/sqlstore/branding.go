package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
)

type EmailBrandingStore struct {
	DB      *sql.DB
	Dialect database.Dialect
}

func (s EmailBrandingStore) ReadEmailBranding(ctx context.Context) (mailer.EmailBranding, error) {
	if s.DB == nil {
		return mailer.EmailBranding{}, errors.New("email branding database is required")
	}
	var value mailer.EmailBranding
	err := s.DB.QueryRowContext(ctx, `SELECT legal_name,physical_address,privacy_policy_url,footer_notice,logo_asset_id,primary_color,email_footer_text FROM mail_branding_settings WHERE singleton_id=1`).Scan(&value.LegalName, &value.PhysicalAddress, &value.PrivacyPolicyURL, &value.FooterNotice, &value.LogoAssetID, &value.PrimaryColor, &value.FooterText)
	if errors.Is(err, sql.ErrNoRows) {
		return mailer.EmailBranding{}, nil
	}
	return value, err
}

func (s EmailBrandingStore) WriteEmailBranding(ctx context.Context, value mailer.EmailBranding) error {
	if s.DB == nil {
		return errors.New("email branding database is required")
	}
	value, err := value.Normalize()
	if err != nil {
		return err
	}
	updated := time.Now().UTC()
	var stamp any = updated
	if s.Dialect == database.SQLite {
		stamp = updated.Format(time.RFC3339Nano)
	}
	query := `INSERT INTO mail_branding_settings(singleton_id,legal_name,physical_address,privacy_policy_url,footer_notice,logo_asset_id,primary_color,email_footer_text,updated_at) VALUES(1,?,?,?,?,?,?,?,?) ON CONFLICT(singleton_id) DO UPDATE SET legal_name=excluded.legal_name,physical_address=excluded.physical_address,privacy_policy_url=excluded.privacy_policy_url,footer_notice=excluded.footer_notice,logo_asset_id=excluded.logo_asset_id,primary_color=excluded.primary_color,email_footer_text=excluded.email_footer_text,updated_at=excluded.updated_at`
	if s.Dialect == database.Postgres {
		query = `INSERT INTO mail_branding_settings(singleton_id,legal_name,physical_address,privacy_policy_url,footer_notice,logo_asset_id,primary_color,email_footer_text,updated_at) VALUES(1,$1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(singleton_id) DO UPDATE SET legal_name=EXCLUDED.legal_name,physical_address=EXCLUDED.physical_address,privacy_policy_url=EXCLUDED.privacy_policy_url,footer_notice=EXCLUDED.footer_notice,logo_asset_id=EXCLUDED.logo_asset_id,primary_color=EXCLUDED.primary_color,email_footer_text=EXCLUDED.email_footer_text,updated_at=EXCLUDED.updated_at`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO mail_branding_settings(singleton_id,legal_name,physical_address,privacy_policy_url,footer_notice,logo_asset_id,primary_color,email_footer_text,updated_at) VALUES(1,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE legal_name=VALUES(legal_name),physical_address=VALUES(physical_address),privacy_policy_url=VALUES(privacy_policy_url),footer_notice=VALUES(footer_notice),logo_asset_id=VALUES(logo_asset_id),primary_color=VALUES(primary_color),email_footer_text=VALUES(email_footer_text),updated_at=VALUES(updated_at)`
	}
	return database.NewTenantTransactor(s.DB, s.Dialect, false).Within(ctx, func(tx database.Executor) error {
		_, err := tx.ExecContext(ctx, query, value.LegalName, value.PhysicalAddress, value.PrivacyPolicyURL, value.FooterNotice, value.LogoAssetID, value.PrimaryColor, value.FooterText, stamp)
		return err
	})
}
