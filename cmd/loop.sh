#!/bin/bash

# Check if correct number of arguments are provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <script_to_execute> <number_of_times>"
    exit 1
fi

SCRIPT=$1
TIMES=$2

# Validate that the script exists and is executable
if [ ! -x "$SCRIPT" ]; then
    echo "Error: $SCRIPT is not executable or does not exist"
    exit 1
fi

# Validate that TIMES is a positive integer
if ! [[ "$TIMES" =~ ^[0-9]+$ ]] || [ "$TIMES" -lt 1 ]; then
    echo "Error: Number of times must be a positive integer"
    exit 1
fi

# Execute the script in a loop without waiting for completion
for ((i=1; i<=$TIMES; i++)); do
    echo "Launching iteration $i of $TIMES"
    ./$SCRIPT &  # The '&' runs the script in the background
    sleep 0.5    # Wait 500ms before launching the next
done

echo "Launched all $TIMES executions of $SCRIPT (running in background)"
