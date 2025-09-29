---
title: AgentHub
---

{{< blocks/cover title="AgentHub: Agent2Agent Communication Broker" image_anchor="top" height="full" >}}
<a class="btn btn-lg btn-primary me-3 mb-4" href="docs/">
  Learn More <i class="fas fa-arrow-alt-circle-right ms-2"></i>
</a>
<a class="btn btn-lg btn-secondary me-3 mb-4" href="https://github.com/owulveryck/agenthub">
  Download <i class="fab fa-github ms-2 "></i>
</a>
<p class="lead mt-5">A unified abstraction library enabling autonomous agents to collaborate through structured task delegation</p>
{{< blocks/link-down color="info" >}}
{{< /blocks/cover >}}


{{% blocks/lead color="primary" %}}
AgentHub is a unified abstraction library that enables autonomous agents to collaborate by exchanging structured tasks through an Agent2Agent communication broker. It provides asynchronous task delegation, flexible agent addressing, and built-in resilience for building complex multi-agent systems.
{{% /blocks/lead %}}


{{% blocks/section color="dark" type="row" %}}
{{% blocks/feature icon="fa-exchange-alt" title="Agent2Agent Communication" %}}
AgentHub implements the Agent2Agent principle, enabling agents to delegate tasks to each other asynchronously with built-in progress tracking and resilience.

Check the [Agent2Agent principle documentation](/docs/explanation/the_agent_to_agent_principle) for more details!
{{% /blocks/feature %}}


{{% blocks/feature icon="fab fa-github" title="Contributions welcome!" url="https://github.com/owulveryck/agenthub" %}}
We do a [Pull Request](https://github.com/owulveryck/agenthub/pulls) contributions workflow on **GitHub**. New users are always welcome!
{{% /blocks/feature %}}


{{% blocks/feature icon="fa-cogs" title="Flexible Architecture" %}}
Build complex multi-agent workflows with our unified abstraction library and comprehensive observability features.

Check the [unified abstraction API reference](/docs/reference/unified_abstraction_api) for more information.
{{% /blocks/feature %}}


{{% /blocks/section %}}


{{% blocks/section %}}
## Documentation Structure
{.h1 .text-center}

Our documentation follows the [Diátaxis Documentation Framework](https://diataxis.fr/), organizing content into tutorials, how-to guides, reference, and explanation.
{.text-center}
{{% /blocks/section %}}


{{% blocks/section type="row" %}}

{{% blocks/feature icon="fa-graduation-cap" title="Tutorials" url="/docs/tutorials/" %}}
Learning-oriented content that takes you through a series of steps to complete a project. Perfect for beginners getting started with AgentHub.

Start with our [Installation and Setup tutorial](/docs/tutorials/installation_and_setup/).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-tools" title="How-To Guides" url="/docs/howto/" %}}
Goal-oriented content that guides you through the steps to solve specific problems and tasks with AgentHub.

Learn how to [create a publisher](/docs/howto/create_publisher/).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-book" title="Reference" url="/docs/reference/" %}}
Information-oriented materials that describe the technical details of AgentHub's components, APIs, and configuration.

Check our [API Reference](/docs/reference/api_reference/).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-lightbulb" title="Explanations" url="/docs/explanation/" %}}
Understanding-oriented content that explains concepts and provides context about how AgentHub works.

Read about the [Broker Architecture](/docs/explanation/broker_architecture/).
{{% /blocks/feature %}}

{{% /blocks/section %}}


{{% blocks/section %}}
## Key Components
{.h1 .text-center}

AgentHub consists of a broker server, publisher agents, subscriber agents, and a unified abstraction library with comprehensive observability features.
{.text-center}
{{% /blocks/section %}}

{{% blocks/section type="row" %}}

{{% blocks/feature icon="fa-server" title="Broker Server" %}}
The central communication hub that routes tasks between agents with built-in resilience and failure handling.

Uses gRPC with Protocol Buffers for efficient communication.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-paper-plane" title="Publisher Agents" %}}
Agents that create and delegate tasks to other agents, monitoring progress and receiving results.

Create your own with the [publisher guide](/docs/howto/create_publisher/).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-inbox" title="Subscriber Agents" %}}
Agents that receive and process tasks from other agents, reporting progress and returning results.

Build subscribers using the [subscriber guide](/docs/howto/create_subscriber/).
{{% /blocks/feature %}}

{{% /blocks/section %}}
