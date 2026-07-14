-- Enable the standard extension to handle automatic generation of UUIDv4 primary keys
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop existing tables if they exist to provide a clean re-initialization script flow
DROP TABLE IF EXISTS trees;
DROP TABLE IF EXISTS estates;

-- 1. Create the Estates Table Profile
CREATE TABLE estates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    width INT NOT NULL,
    length INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Enforce the assignment size ceiling restriction (from 1 to 50,000 plots inclusive)
    CONSTRAINT chk_estate_width CHECK (width >= 1 AND width <= 50000),
    CONSTRAINT chk_estate_length CHECK (length >= 1 AND length <= 50000)
);

-- 2. Create the Trees Table Profile
CREATE TABLE trees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    estate_id UUID NOT NULL,
    x INT NOT NULL,
    y INT NOT NULL,
    height INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Maintain coordinate limits to positive numbers
    CONSTRAINT chk_tree_x CHECK (x >= 1),
    CONSTRAINT chk_tree_y CHECK (y >= 1),
    
    -- Enforce assignment tree height ceilings (from 1 to 30 meters inclusive)
    CONSTRAINT chk_tree_height CHECK (height >= 1 AND height <= 30),
    
    -- Establish relationship linkage back to the master estate node
    CONSTRAINT fk_tree_estate FOREIGN KEY (estate_id) 
        REFERENCES estates(id) ON DELETE CASCADE,
        
    -- Spatial Unique Constraint: Prevents a plot from containing duplicate trees
    CONSTRAINT uq_estate_plot_coordinate UNIQUE (estate_id, x, y)
);

-- 3. Database Performance Tuning Optimization Indexes
-- Indexes for foreign key lookups and analytics sorting calculations (Min/Max/Median)
CREATE INDEX idx_trees_estate_id ON trees(estate_id);
CREATE INDEX idx_trees_height ON trees(estate_id, height);

-- Composite coverage index optimization for high performance on coordinate lookups
CREATE INDEX idx_trees_coordinates ON trees(estate_id, x, y);
