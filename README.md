# Asynchronous state machine

## Overview

This module implements an asynchronous state machine that adheres to the RTC (Run to Completion) execution model. The state machine ensures that incoming events are buffered and executed in order. It also provides the capability to wait for the completion of state transitions using the result channel provided by the Trigger method.

## Features

-   Asynchronous Execution: Handles state transitions asynchronously, allowing other processes to continue without waiting.
-   Sequential Execution: Waiting for a transition to complete using the result channel returned by the Trigger method.
-   RTC Model Compliance: Adheres to the Run-to-Completion (RTC) model, ensuring that each event is fully processed before the next event is handled.
-   Event Buffering: Buffers incoming events to guarantee that they are executed in the order they are received.
