# -----------------------------------------------------------------------------
# Monitoring Stack (Optional)
# Deploys Prometheus and Grafana when var.enable_monitoring is true
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# CloudWatch Log Groups for Monitoring
# -----------------------------------------------------------------------------

resource "aws_cloudwatch_log_group" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  name              = "/ecs/${local.name_prefix}/prometheus"
  retention_in_days = 14

  tags = {
    Name = "${local.name_prefix}-prometheus-logs"
  }
}

resource "aws_cloudwatch_log_group" "grafana" {
  count = var.enable_monitoring ? 1 : 0

  name              = "/ecs/${local.name_prefix}/grafana"
  retention_in_days = 14

  tags = {
    Name = "${local.name_prefix}-grafana-logs"
  }
}

# -----------------------------------------------------------------------------
# Prometheus Task Definition
# -----------------------------------------------------------------------------

resource "aws_ecs_task_definition" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  family                   = "${local.name_prefix}-prometheus"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "prometheus"
      image     = "prom/prometheus:latest"
      essential = true

      portMappings = [
        {
          containerPort = 9090
          protocol      = "tcp"
          name          = "prometheus"
        }
      ]

      command = [
        "--config.file=/etc/prometheus/prometheus.yml",
        "--storage.tsdb.path=/prometheus",
        "--web.console.libraries=/usr/share/prometheus/console_libraries",
        "--web.console.templates=/usr/share/prometheus/consoles"
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.prometheus[0].name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "prometheus"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "wget -q --spider http://localhost:9090/-/healthy || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 30
      }
    }
  ])

  tags = {
    Name = "${local.name_prefix}-prometheus-task"
  }
}

# -----------------------------------------------------------------------------
# Grafana Task Definition
# -----------------------------------------------------------------------------

resource "aws_ecs_task_definition" "grafana" {
  count = var.enable_monitoring ? 1 : 0

  family                   = "${local.name_prefix}-grafana"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "grafana"
      image     = "grafana/grafana:latest"
      essential = true

      portMappings = [
        {
          containerPort = 3000
          protocol      = "tcp"
          name          = "grafana"
        }
      ]

      environment = [
        {
          name  = "GF_SECURITY_ADMIN_PASSWORD"
          value = var.grafana_admin_password
        },
        {
          name  = "GF_SERVER_HTTP_PORT"
          value = "3000"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.grafana[0].name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "grafana"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "wget -q --spider http://localhost:3000/api/health || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 30
      }
    }
  ])

  tags = {
    Name = "${local.name_prefix}-grafana-task"
  }
}

# -----------------------------------------------------------------------------
# ALB Target Groups for Monitoring
# -----------------------------------------------------------------------------

resource "aws_lb_target_group" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  name        = "${local.name_prefix}-prom-tg"
  port        = 9090
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/-/healthy"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 3
  }

  tags = {
    Name = "${local.name_prefix}-prometheus-tg"
  }
}

resource "aws_lb_target_group" "grafana" {
  count = var.enable_monitoring ? 1 : 0

  name        = "${local.name_prefix}-grafana-tg"
  port        = 3000
  protocol    = "HTTP"
  vpc_id      = aws_vpc.main.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/api/health"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 3
  }

  tags = {
    Name = "${local.name_prefix}-grafana-tg"
  }
}

# -----------------------------------------------------------------------------
# ALB Listener Rules for Monitoring
# -----------------------------------------------------------------------------

resource "aws_lb_listener_rule" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  listener_arn = aws_lb_listener.http.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.prometheus[0].arn
  }

  condition {
    path_pattern {
      values = ["/prometheus*"]
    }
  }
}

resource "aws_lb_listener_rule" "grafana" {
  count = var.enable_monitoring ? 1 : 0

  listener_arn = aws_lb_listener.http.arn
  priority     = 101

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.grafana[0].arn
  }

  condition {
    path_pattern {
      values = ["/grafana*"]
    }
  }
}

# -----------------------------------------------------------------------------
# ECS Services for Monitoring
# -----------------------------------------------------------------------------

resource "aws_ecs_service" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  name            = "${local.name_prefix}-prometheus"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.prometheus[0].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.monitoring[0].id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.prometheus[0].arn
    container_name   = "prometheus"
    container_port   = 9090
  }

  tags = {
    Name = "${local.name_prefix}-prometheus-service"
  }
}

resource "aws_ecs_service" "grafana" {
  count = var.enable_monitoring ? 1 : 0

  name            = "${local.name_prefix}-grafana"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.grafana[0].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.monitoring[0].id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.grafana[0].arn
    container_name   = "grafana"
    container_port   = 3000
  }

  tags = {
    Name = "${local.name_prefix}-grafana-service"
  }
}
