### route

The routing service handles passing requests to the provider endpoints.

### Status

The current implementation draws on a lot of work done by the GOV.UK team in their router.
They [wrote](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk/) about
some of their experiences and shared their progress, much of which is the basis for this. The muxer has been
included and modified, however, to allow for logging.

Logging is handled by Apache Kafka. To run this you should set up the Kafka server. Topics should be created for you.
The sarama library handles interfacing with Kafka and its protocol.

Route definitions are defined in a simple way in the `hosts.conf` file.
