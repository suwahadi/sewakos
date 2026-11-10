-- +goose Up
CREATE INDEX idx_properties_area_id ON properties (area_id);
CREATE INDEX idx_properties_lat_lng ON properties (latitude, longitude);

-- +goose Down
DROP INDEX idx_properties_lat_lng;
DROP INDEX idx_properties_area_id;
