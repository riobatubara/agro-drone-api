# agro-drone-api

### Use Case: Map Plantation Estates and Record Palm Tree Layouts

#### 1. Brief Description
Enables plantation operators to digitally model their physical agricultural land into standard geometric grid units and record the exact spatial location and physical height of individual palm trees for downstream drone inspection mapping.

#### 2. Actors
- **Plantation Manager / Operator**: The primary user who inputs the physical layout configurations of the field.

- **Plantation Management System**: The backend application that validates, structures, and persists the geospatial plantation data.

#### 3. Preconditions
The operator has the physical land survey data containing the overall dimensions of the plantation estate (p. 1).The operator knows the coordinate positions and estimated heights of the planted palm trees.

#### 4. Basic Flow of Events (Plantation Modeling)
 **1. Define Estate Grid Dimensions**: The operator enters the absolute length (West to East) and width (South to North) of a square or rectangular plantation estate.
 
 **2. Generate Plot Breakdown**: The system automatically divides the defined estate into distinct, adjacent 10 × 10 meter square plot blocks.
 
 **3. Log Palm Tree Asset**: The operator selects a specific plot using grid coordinates (x, y) and records a palm tree.
 
 **4. Assign Tree Attributes**: The operator specifies the integer height of the palm tree.
 
 **5. Validate Layout Entry**: The system maps the tree's position and confirms the asset is securely saved in the database.

#### 5. Visual System Blueprints

##### Spatial Modeling & 2D Grid Layout Blueprint
This diagram shows how a physical estate (e.g., 60m × 30m) is segmented into 10-meter plots with palm trees positioned precisely in the center of their respective coordinate blocks.
<br>

```text
   NORTH ▲
         │
       3 ├───────┬───────┬───────┬───────┬───────┬───────┐
         │       │       │       │       │ ●     │       │
         │       │       │       │       │ (5, 3)│       │
       2 ├───────┬───────┼───────┼───────┼───────┼───────┤
         │       │       │ ●     │ ●     │       │ ●     │
         │       │       │ (3, 2)│ (4, 2)│       │ (6, 2)│
 width 1 ├───────┬───────┼───────┼───────┼───────┼───────┤
         │       │       │ ●     │       │       │       │
         │       │       │ (3, 1)│       │       │       │
         └───────┴───────┴───────┴───────┴───────┴───────┴► EAST
             1       2       3       4       5       6
          ◄─────────────────── length ───────────────────►
```


##### Drone Flight Path & Vertical Altitude Trajectory
The flight blueprint details the dynamic altitude adjustments required to scan the field. The drone flies **exactly 1 meter above the ground or tree canopy** and moves using single-axis transitions.

```text
   NORTH ▲
         │
         ├────────┬────────┬────────┬────────┬────────┬────────┐
         │        │        │        │        │        │        │
       3 │   ▲ ────────────────────────────────────────────►   │
         │   │    │        │        │        │        │        │
         ├───│────┬────────┼────────┼────────┼────────┼────────┤
         │   │    │        │        │        │        │        │
       2 │   ◄─────────────────────────────────────────── ▲    │
         │        │        │        │        │        │   │    │
         ├────────┬────────┼────────┼────────┼────────┼───│────┤
         │        │        │        │        │        │   │    │
       1 │    ────────────────────────────────────────────►    │
         │        │        │        │        │        │        │
         └────────┴────────┴────────┴────────┴────────┴────────┴► EAST
              1        2        3        4        5       6
```

```text
 ALTITUDE (m)
     ▲
  6  │         ┌───────┐
  5  │ ┌───────┘ 1m cls │               ┌───────┐
  4  │ │ 1m cls         │ ┌───────┐     │ 1m cls│
  3  │ │                └─┘ 1m cls│     │       │
  2  │ │                          └─────┘       │
  1  │ │ 1m clearance above empty plots         │
  0  └─┴─┬─────┬─┬─────┬─┬─────┬─┬─────┬─┬─────┬┴─► FLIGHT PATH
        (1,1)   (2,1)   (3,1)   (4,1)   (5,1)
         ▲       │       │       │       │
      [Takeoff] [5m]    [3m]    [4m]    [Empty]
                Tree    Tree    Tree    [Landing]
```