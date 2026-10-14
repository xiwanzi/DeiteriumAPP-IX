CREATE TABLE IF NOT EXISTS identity_profile_resolutions (
 case_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 target_user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 profile_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 previous_game_id VARCHAR(32) NOT NULL,
 claimant_email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 evidence_type VARCHAR(24) CHARACTER SET ascii NOT NULL,
 evidence_reference VARCHAR(240) NOT NULL,
 game_verification_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 reason VARCHAR(500) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 claimed_user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 claimed_at DATETIME(6) NULL,
 INDEX ix_profile_resolution_target(target_user_id,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS identity_profile_reservation_edits (
 edit_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 case_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 previous_email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 next_email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 reason VARCHAR(500) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 INDEX ix_reservation_edit_case(case_id,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS identity_profile_reservations (
 profile_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 case_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 claimant_email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 claimed_user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 claimed_at DATETIME(6) NULL,
 INDEX ix_profile_reservation_case(case_id)
) ENGINE=InnoDB;
