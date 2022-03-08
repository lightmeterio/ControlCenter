// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

const (
	// NOTE: those values are used as constants in the database
	// therefore never change their order or remove elements.
	// you can add new elements in the end, before "lastKey", though.
	//nolint
	firstResultKey = iota

	ConnectionBeginKey
	ConnectionEndKey
	ConnectionClientHostnameKey
	ConnectionClientIPKey

	QueueBeginKey
	QueueEndKey
	QueueSenderLocalPartKey
	QueueSenderDomainPartKey
	QueueOriginalMessageSizeKey
	QueueProcessedMessageSizeKey
	QueueNRCPTKey
	QueueMessageIDKey
	QueueDeliveryNameKey

	ResultDeliveryTimeKey
	ResultRecipientLocalPartKey
	ResultRecipientDomainPartKey
	ResultOrigRecipientLocalPartKey
	ResultOrigRecipientDomainPartKey
	ResultDelayKey
	ResultDelaySMTPDKey
	ResultDelayCleanupKey
	ResultDelayQmgrKey
	ResultDelaySMTPKey
	ResultDSNKey
	ResultStatusKey
	ResultDeliveryFilenameKey
	ResultDeliveryFileLineKey
	ResultRelayNameKey
	ResultRelayIPKey
	ResultRelayPortKey
	ResultDeliveryServerKey
	ResultMessageDirectionKey

	PickupUidKey
	PickupSenderKey

	ConnectionFilenameKey
	ConnectionLineKey
	DisconnectionFilenameKey
	DisconnectionLineKey
	QueueFilenameKey
	QueueLineKey
	QueueCommitFilenameKey
	QueueCommitLineKey
	MessageIdFilenameKey
	MessageIdLineKey

	MessageIdIsCorruptedKey
	MessageExpiredTime
	ParentQueueDeliveryNameKey

	ResultDeliveryLineChecksum

	lasResulttKey
)

var (
	KeysToLabels = map[int]string{
		ConnectionBeginKey:          "conn_ts_begin",
		ConnectionEndKey:            "conn_ts_end",
		ConnectionClientHostnameKey: "client_hostname",
		ConnectionClientIPKey:       "client_ip",

		QueueBeginKey:                "queue_ts_begin",
		QueueEndKey:                  "queue_ts_end",
		QueueSenderLocalPartKey:      "sender_local_part",
		QueueSenderDomainPartKey:     "sender_domain_part",
		QueueOriginalMessageSizeKey:  "orig_size",
		QueueProcessedMessageSizeKey: "processed_size",
		QueueNRCPTKey:                "nrcpt",
		QueueMessageIDKey:            "message_id",
		QueueDeliveryNameKey:         "delivery_queue",

		ResultDeliveryTimeKey:            "delivery_ts",
		ResultRecipientLocalPartKey:      "recipient_local_part",
		ResultRecipientDomainPartKey:     "recipient_domain_part",
		ResultOrigRecipientLocalPartKey:  "orig_recipient_local_part",
		ResultOrigRecipientDomainPartKey: "orig_recipient_domain_part",
		ResultDelayKey:                   "delay",
		ResultDelaySMTPDKey:              "delay_smtpd",
		ResultDelayCleanupKey:            "delay_cleanup",
		ResultDelayQmgrKey:               "delay_qmgr",
		ResultDelaySMTPKey:               "delay_smtp",
		ResultDSNKey:                     "dsn",
		ResultStatusKey:                  "status",
		ResultDeliveryFilenameKey:        "delivery_filename",
		ResultDeliveryFileLineKey:        "delivery_line",
		ResultRelayNameKey:               "relay_name",
		ResultRelayIPKey:                 "relay_ip",
		ResultRelayPortKey:               "relay_port",
		ResultDeliveryServerKey:          "delivery_server",
		ResultMessageDirectionKey:        "message_direction",

		PickupUidKey:    "pickup_uid",
		PickupSenderKey: "pickup_sender",

		ConnectionFilenameKey:      "connection_filename",
		ConnectionLineKey:          "connection_line",
		DisconnectionFilenameKey:   "disconnection_filename",
		DisconnectionLineKey:       "disconnection_line",
		QueueFilenameKey:           "queue_filename",
		QueueLineKey:               "queue_line",
		QueueCommitFilenameKey:     "queue_commit_filename",
		QueueCommitLineKey:         "queue_commit_line",
		MessageIdFilenameKey:       "messageid_filename",
		MessageIdLineKey:           "messageid_line",
		MessageIdIsCorruptedKey:    "messageid_is_corrupted",
		MessageExpiredTime:         "message_expiration_time",
		ParentQueueDeliveryNameKey: "parent_delivery_queue",
	}
)
