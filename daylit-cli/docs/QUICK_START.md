# Quick Start

```bash
# Initialize daylit
daylit init

# Add some tasks
daylit task add "Morning prayer" --duration 30 --recurrence daily --earliest 07:00 --latest 09:00

daylit task add "Deep work" --duration 90 --recurrence n_days --interval 2 --earliest 09:00 --latest 13:00 --priority 1

daylit task add "Gym" --duration 60 --recurrence weekly --weekdays mon,wed,fri --earliest 14:00 --latest 18:00

# Add an appointment
daylit task add "Team meeting" --duration 60 --fixed-start 10:00 --fixed-end 11:00

# Generate today's plan
daylit plan today

# Check what you should be doing now
daylit now

# Give feedback on completed tasks
daylit feedback --rating on_track --note "Went well"

# View today's full plan
daylit day today
```
