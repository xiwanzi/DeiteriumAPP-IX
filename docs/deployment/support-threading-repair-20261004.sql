-- One-time repair, service stopped, database backed up. All messages remain.
DELIMITER $$
BEGIN NOT ATOMIC
 DECLARE EXIT HANDLER FOR SQLEXCEPTION BEGIN ROLLBACK; RESIGNAL; END;
 START TRANSACTION;
 IF (SELECT COUNT(*) FROM support_tickets a JOIN support_tickets b ON a.requester_email=b.requester_email WHERE a.ticket_id='202610047494' AND b.ticket_id='202610049962' AND a.source='EMAIL' AND b.source='ADMIN' AND a.status='CLOSED' AND b.status='CLOSED')<>1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Ticket pair or status changed; recheck required'; END IF;
 IF NOT EXISTS(SELECT 1 FROM support_messages WHERE ticket_id='202610047494' AND direction='INBOUND' AND body LIKE '%202610049962%') THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Original ticket evidence missing'; END IF;
 INSERT INTO support_ticket_aliases VALUES('202610047494','202610049962',UTC_TIMESTAMP(6));
 UPDATE support_messages SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE support_attachments SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE support_mail_links SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE support_inbound SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE support_submissions SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE support_ticket_aliases SET ticket_id='202610049962' WHERE ticket_id='202610047494';
 UPDATE mail_outbox SET reference_id='202610049962' WHERE reference_id='202610047494' AND kind IN ('TICKET_CREATED','TICKET_REPLY');
 UPDATE support_tickets a JOIN support_tickets b ON b.ticket_id='202610047494' SET a.status='CLOSED',a.version=GREATEST(a.version,b.version)+1,a.updated_at=GREATEST(a.updated_at,b.updated_at) WHERE a.ticket_id='202610049962';
 DELETE FROM support_tickets WHERE ticket_id='202610047494';
 IF (SELECT COUNT(*) FROM support_tickets a JOIN support_tickets b ON a.requester_email=b.requester_email WHERE a.ticket_id='202610046363' AND b.ticket_id='202610040432' AND a.source='EMAIL' AND b.source='ADMIN' AND a.status='CLOSED' AND b.status='CLOSED')<>1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Ticket pair or status changed; recheck required'; END IF;
 IF NOT EXISTS(SELECT 1 FROM support_messages WHERE ticket_id='202610046363' AND direction='INBOUND' AND body LIKE '%202610040432%') THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Original ticket evidence missing'; END IF;
 INSERT INTO support_ticket_aliases VALUES('202610046363','202610040432',UTC_TIMESTAMP(6));
 UPDATE support_messages SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE support_attachments SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE support_mail_links SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE support_inbound SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE support_submissions SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE support_ticket_aliases SET ticket_id='202610040432' WHERE ticket_id='202610046363';
 UPDATE mail_outbox SET reference_id='202610040432' WHERE reference_id='202610046363' AND kind IN ('TICKET_CREATED','TICKET_REPLY');
 UPDATE support_tickets a JOIN support_tickets b ON b.ticket_id='202610046363' SET a.status='CLOSED',a.version=GREATEST(a.version,b.version)+1,a.updated_at=GREATEST(a.updated_at,b.updated_at) WHERE a.ticket_id='202610040432';
 DELETE FROM support_tickets WHERE ticket_id='202610046363';
 IF (SELECT COUNT(*) FROM support_tickets a JOIN support_tickets b ON a.requester_email=b.requester_email WHERE a.ticket_id='202610046121' AND b.ticket_id='202610045757' AND a.source='EMAIL' AND b.source='ADMIN' AND a.status='OPEN' AND b.status='WAITING_CUSTOMER')<>1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Ticket pair or status changed; recheck required'; END IF;
 IF NOT EXISTS(SELECT 1 FROM support_messages WHERE ticket_id='202610046121' AND direction='INBOUND' AND body LIKE '%202610045757%') THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Original ticket evidence missing'; END IF;
 INSERT INTO support_ticket_aliases VALUES('202610046121','202610045757',UTC_TIMESTAMP(6));
 UPDATE support_messages SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE support_attachments SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE support_mail_links SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE support_inbound SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE support_submissions SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE support_ticket_aliases SET ticket_id='202610045757' WHERE ticket_id='202610046121';
 UPDATE mail_outbox SET reference_id='202610045757' WHERE reference_id='202610046121' AND kind IN ('TICKET_CREATED','TICKET_REPLY');
 UPDATE support_tickets a JOIN support_tickets b ON b.ticket_id='202610046121' SET a.status='OPEN',a.version=GREATEST(a.version,b.version)+1,a.updated_at=GREATEST(a.updated_at,b.updated_at) WHERE a.ticket_id='202610045757';
 DELETE FROM support_tickets WHERE ticket_id='202610046121';
 COMMIT;
END$$
DELIMITER ;
