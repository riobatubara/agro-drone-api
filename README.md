# agro-drone-api

### A. Use Case: Map Plantation Estates and Record Palm Tree Layouts

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
         ├───────┬───────┬───────┬───────┬───────┬───────┐
       3 │       │       │       │       │ ●     │       │
         │       │       │       │       │ (5, 3)│       │
         ├───────┬───────┼───────┼───────┼───────┼───────┤
       2 │       │       │ ●     │ ●     │       │ ●     │
         │       │       │ (3, 2)│ (4, 2)│       │ (6, 2)│
         ├───────┬───────┼───────┼───────┼───────┼───────┤
 width 1 │       │       │ ●     │       │       │       │
         │       │       │ (3, 1)│       │       │       │
         └───────┴───────┴───────┴───────┴───────┴───────┴──► EAST
             1       2       3       4       5       6
          ◄─────────────────── length ───────────────────►
```


##### Drone Flight Path & Vertical Altitude Trajectory
The flight blueprint details the dynamic altitude adjustments required to scan the field. The drone flies **exactly 1 meter above the ground or tree canopy** and moves using single-axis transitions.

###### Horizontal Sweeping Pattern (Top-Down Boustrophedon View)
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
         └────────┴────────┴────────┴────────┴────────┴────────┴──► EAST
              1        2        3        4        5       6
```

###### Vertical Profile Clearance & Takeoff / Landing Vectors (Side Cross-Section View)
A 5×1 plot row containing three trees of heights 5m, 3m, and 4m respectively. Total computed distance = 54 meters (40m horizontal + 14m vertical).

```text
            10m             10m
     ───────────────► ───────────────►           10m              10m
     ▲     ▲                         │     ───────────────► ───────────────►
     │     │                         │     ▲                          ▲     │
     │  1m │                      2m │  1m │                       1m │     │
     │     ▼   ┌───────────┐         │     |                          │     │
     │     ▲   | Palm Tree |  ▲      ▼               ┌───────────┐    ▼     │
     │     │   └──       ──┘  │                      | Palm Tree |    ▲  4m │         
  5m │     │      │     │  1m │                      └──       ──┘    │     │
     │     │      │     │     ▼   ┌───────────┐         │     │       │     │
     │     │      │     │     ▲   │ Palm Tree |         │     │       │     │
     │  5m │      │     │     │   └──       ──┘         │     │    4m │     ▼
     ▲     │      │     │  3m │      │     │            │     │       │  1m │
 1m  │     │      │     │     │      │     │            │     │       │     │
     │     │      │     │     │      │     │            │     │       │     ▼
───────────┬──────────────────┬────────────────────┬──────────────────┬───────────┬
first plot                                                              last plot
                                total distance = 54
```

#### 6. Domain Rules & Core Constraints
| Rule Component | Requirement Details | Technical Clarification |
| :--- | :--- | :--- |
| **Estate Geometry** | Must always be perfectly square or rectangular. | Irregularly shaped borders or organic perimeters are not supported. |
| **Spatial Unit** | Standardized to perfect 10 × 10 m² plots. | Dimensions must be provided in 10-meter increments (1 to 50000 range). |
| **Tree Heights** | Must range from **1 to 30 meters** inclusive. | Heights must be stored strictly as integers due to uniform crop varieties. |
| **Flight Pathing** | Single-axis movement at any given time. | Drone moves strictly vertical, East-West, or North-South. Diagonal flights are locked out. |
| **Target Proximity** | Maintain exactly 1 meter of space above target. | Drone passes 1m above empty soil (1m altitude) or 1m above the tree apex (Height + 1). |


#### 7. Exception Flows (System Validations)

##### Exception Flow 1: Out of Bounds Dimensions
* **Condition**: The operator inputs size metrics below 1 or over 50,000 plots.
* **System Action**: The system cancels processing and returns an HTTP `400 Bad Request`.

##### Exception Flow 2: Asset Overlap Error
* **Condition**: The operator logs a tree on a coordinate index that already contains a tracked item.
* **System Action**: The mutation request is blocked, throwing an HTTP `400 Bad Request` validation error.

##### Exception Flow 3: Battery Life Exhaustion
* **Condition**: Calculated navigation exceeds the requested maximum battery range threshold (`max_distance`).
* **System Action**: The system tracks the path limit step-by-step, intercepts path calculation at depletion, and logs the precise `(x, y)` coordinate where the drone performs a forced touchdown.

------