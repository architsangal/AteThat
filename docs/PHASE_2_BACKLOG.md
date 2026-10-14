# Phase 2 backlog

## Terms and Conditions: Phase 1 data-recovery limitation

### Phase 1 decision

Phase 1 uses MongoDB Atlas M0 without application-managed backups, snapshots, or point-in-time recovery. Recovery after accidental deletion, application defects, account compromise, or an Atlas data-loss event is not guaranteed. This is an explicitly accepted Phase 1 risk for the initial low-risk release.

### Phase 2 requirement

Before publishing or materially revising the public Terms and Conditions, define and approve clear user-facing wording for the data-recovery limitation.

The wording must:

- accurately describe the recovery posture that is actually in effect at publication time;
- avoid promising backup, restoration, recovery-point, or recovery-time guarantees unless those capabilities have been implemented and verified;
- be reviewed as a legal/product policy change before publication; and
- be updated if the product adopts Atlas backups, point-in-time recovery, or an application-managed backup process.

### Acceptance criteria

- The approved Terms and Conditions link is available from the production application.
- The published wording matches the current backup and recovery implementation.
- Any implemented recovery commitment has a documented owner, recovery procedure, and test record.
