-- Insert processed results into output table
INSERT INTO output_results
SELECT event_type, user_id, event_count, window_start, window_end
FROM processed_events;