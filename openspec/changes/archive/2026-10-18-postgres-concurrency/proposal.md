# Independent PostgreSQL conflict evidence

Exercise the existing adapter through two independently connected pools, with concurrent callers released by a channel barrier. Prove atomic replay, changed-payload conflict, stale-version conflict and single reversal without relying on synchronization sleeps. Add PostgreSQL timestamp-tie pagination evidence. No API change is proposed.
