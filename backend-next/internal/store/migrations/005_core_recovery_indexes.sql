SET @dc_index_ddl = (SELECT IF(COUNT(*)=0,'CREATE INDEX ix_core_operation_expiry ON core_operations(state,expires_at)','DO 0') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='core_operations' AND INDEX_NAME='ix_core_operation_expiry');
PREPARE dc_index_statement FROM @dc_index_ddl;
EXECUTE dc_index_statement;
DEALLOCATE PREPARE dc_index_statement;
SET @dc_index_ddl = NULL;
