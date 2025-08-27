-- Create processing view with windowed aggregation
CREATE TABLE processed_events AS
SELECT 
  event_type,
  user_id,
  COUNT(*) as event_count,
  TUMBLE_START(ts, INTERVAL '1' SECOND) as window_start,
  TUMBLE_END(ts, INTERVAL '1' SECOND) as window_end
FROM input_events
GROUP BY 
  event_type,
  user_id,
  TUMBLE(ts, INTERVAL '1' SECOND);
